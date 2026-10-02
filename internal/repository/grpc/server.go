package rpc

import (
	"context"
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

const queryTimeout = 30 * time.Second

// Server is the master's control core: spec repo, status repo, and the
// routing table of live slave streams. Plain methods (NodeSpec, SetNodeSpec,
// Statuses, RouteQuery) serve the REST layer; Watch/ReportStatus are the
// slave transport.
type Server struct {
	UnimplementedControlServer
	specs  interfaces.SpecRepository
	status interfaces.StatusRepository

	mu      sync.Mutex
	streams map[string]*nodeStream
}

type nodeStream struct {
	mu      sync.Mutex
	stream  grpc.BidiStreamingServer[StreamMsg, StreamMsg]
	nextID  int64
	waiters map[int64]chan *QueryResult
}

func NewServer(specs interfaces.SpecRepository, status interfaces.StatusRepository) *Server {
	return &Server{specs: specs, status: status, streams: map[string]*nodeStream{}}
}

// Watch binds a slave by the node in its hello, pushes spec changes, and
// relays query answers back to waiters.
func (s *Server) Watch(stream grpc.BidiStreamingServer[StreamMsg, StreamMsg]) error {
	hello, err := stream.Recv()
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
	if err := sendSpec(); err != nil {
		return err
	}

	go func() {
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
				return err
			}
		case <-streamCtx.Done():
			return nil
		}
	}
}

func (s *Server) ReportStatus(ctx context.Context, st *Status) (*Empty, error) {
	if err := s.status.Report(ctx, entity.NodeStatus{
		Node: st.Node, State: st.State, Detail: st.Detail,
	}); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// REST-facing operations (plain methods, no wire protocol).

func (s *Server) NodeSpec(ctx context.Context, node string) (*entity.Spec, error) {
	return s.specs.Get(ctx, node)
}

// SetNodeSpec persists an assignment and publishes it to the node's stream.
func (s *Server) SetNodeSpec(ctx context.Context, spec *entity.Spec) error {
	return s.specs.Set(ctx, spec)
}

func (s *Server) Statuses(ctx context.Context) ([]entity.NodeStatus, error) {
	return s.status.List(ctx)
}

// RouteQuery sends a query over the node's stream and waits for the answer.
func (s *Server) RouteQuery(ctx context.Context, node, engine, sql string) (string, error) {
	s.mu.Lock()
	ns := s.streams[node]
	s.mu.Unlock()
	if ns == nil {
		return "", errors.New("node " + node + " is not connected")
	}
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

	if err := ns.stream.Send(&StreamMsg{Payload: &StreamMsg_Query{Query: &QueryRequest{Id: id, Engine: engine, Sql: sql}}}); err != nil {
		return "", errors.New("node " + node + " is not connected")
	}
	select {
	case r := <-ch:
		if r.Error != "" {
			return "", errors.New(r.Error)
		}
		return r.Output, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(queryTimeout):
		return "", errors.New("query timed out")
	}
}

func (ns *nodeStream) deliver(r *QueryResult) {
	ns.mu.Lock()
	ch := ns.waiters[r.Id]
	ns.mu.Unlock()
	select {
	case ch <- r:
	default:
	}
}

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
