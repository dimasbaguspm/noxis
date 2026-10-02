package rpc

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"

	"noxis/internal/interfaces"
)

// queryTimeout bounds a routed query end to end.
const queryTimeout = 30 * time.Second

// Server implements the Control service on the master. Stateless beyond the
// repositories: desired assignments (what should run) and the status echo
// (what nodes report running). Actual runtime never lives here.
type Server struct {
	UnimplementedControlServer
	specs  interfaces.SpecRepository
	status interfaces.StatusRepository

	mu      sync.Mutex
	streams map[string]*nodeStream // node id -> live Watch stream
}

func NewServer(specs interfaces.SpecRepository, status interfaces.StatusRepository) *Server {
	return &Server{
		specs:   specs,
		status:  status,
		streams: map[string]*nodeStream{},
	}
}

// nodeStream is one slave's bidirectional Watch river: master pushes specs,
// slave answers queries. Waiter channels correlate user queries to results.
type nodeStream struct {
	mu      sync.Mutex
	stream  grpc.BidiStreamingServer[StreamMsg, StreamMsg]
	nextID  int64
	waiters map[int64]chan *QueryResult
}

func (ns *nodeStream) query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	ns.mu.Lock()
	ns.nextID++
	id := ns.nextID
	ch := make(chan *QueryResult, 1)
	ns.waiters[id] = ch
	ns.mu.Unlock()

	defer func() {
		ns.mu.Lock()
		delete(ns.waiters, id)
		ns.mu.Unlock()
	}()

	req.Id = id
	if err := ns.stream.Send(&StreamMsg{Payload: &StreamMsg_Query{Query: req}}); err != nil {
		return nil, errors.New("node stream is not connected")
	}
	select {
	case r := <-ch:
		return r, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(queryTimeout):
		return nil, errors.New("query timed out")
	}
}

// delivers routes a slave's answer to the waiting caller.
func (ns *nodeStream) deliver(r *QueryResult) {
	ns.mu.Lock()
	ch := ns.waiters[r.Id]
	ns.mu.Unlock()
	select {
	case ch <- r:
	default: // nobody waiting anymore
	}
}

// reg / unreg keep the routing table of live streams.
func (s *Server) reg(node string, ns *nodeStream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[node] = ns
}

func (s *Server) unreg(node string, ns *nodeStream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streams[node] == ns {
		delete(s.streams, node)
	}
}

// Query routes over the node's Watch stream. The stream sees no packets it
// cannot answer: spec pushes one way, query round-trips the other.
func (s *Server) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	s.mu.Lock()
	ns := s.streams[req.Node]
	s.mu.Unlock()
	if ns == nil {
		return nil, errors.New("node " + req.Node + " is not connected")
	}
	return ns.query(ctx, req)
}

func (s *Server) GetDesiredState(ctx context.Context, req *NodeId) (*DesiredSpec, error) {
	spec, err := s.specs.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return SpecToProto(spec), nil
}

func (s *Server) SetDesiredState(ctx context.Context, req *SetDesiredStateRequest) (*Empty, error) {
	spec := SpecFromProto(req.Spec)
	spec.Node = req.Node // node rides at the request level, not inside DesiredSpec
	if err := s.specs.Set(ctx, spec); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// Watch binds a slave's stream by the node in its first hello message, then
// pushes spec on every change and processes query results until disconnect.
func (s *Server) Watch(stream grpc.BidiStreamingServer[StreamMsg, StreamMsg]) error {
	hello, err := stream.Recv() // the slave identifies itself first
	if err != nil {
		return err
	}
	node := hello.Node
	if node == "" {
		return errors.New("stream hello must carry a node")
	}

	ns := &nodeStream{stream: stream, waiters: map[int64]chan *QueryResult{}}
	s.reg(node, ns)
	defer s.unreg(node, ns)

	streamCtx := stream.Context()
	ch := s.specs.Subscribe(streamCtx, node)
	defer s.specs.Unsubscribe(streamCtx, node, ch)

	sendSpec := func() error {
		spec, err := s.specs.Get(streamCtx, node)
		if err != nil {
			return err
		}
		return stream.Send(&StreamMsg{Node: node, Payload: &StreamMsg_Spec{Spec: SpecToProto(spec)}})
	}
	if err := sendSpec(); err != nil { // hello snapshot: full replay on (re)connect
		return err
	}

	go func() { // recv pump: query answers come back this way
		for {
			msg, err := stream.Recv()
			if err != nil {
				return
			}
			if r := msg.GetResult(); r != nil {
				ns.deliver(r)
			}
		}
	}()

	for {
		select {
		case <-ch:
			if err := sendSpec(); err != nil {
				slog.WarnContext(streamCtx, "watch send failed", "node", node, "error", err)
				return err
			}
		case <-streamCtx.Done():
			return nil
		}
	}
}

func (s *Server) ReportStatus(ctx context.Context, st *Status) (*Empty, error) {
	if err := s.status.Report(ctx, statusFromProto(st)); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

func (s *Server) ListStatus(ctx context.Context, _ *Empty) (*StatusList, error) {
	rows, err := s.status.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Status, 0, len(rows))
	for _, r := range rows {
		out = append(out, statusToProto(r))
	}
	return &StatusList{Statuses: out}, nil
}
