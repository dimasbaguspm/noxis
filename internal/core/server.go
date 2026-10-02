package core

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"

	"noxis/internal/store"
)

// Server implements the Control service on the master. It keeps no state of
// its own beyond the SQLite store.
type Server struct {
	UnimplementedControlServer
	st *store.Store
}

func NewServer(st *store.Store) *Server { return &Server{st: st} }

func (s *Server) GetDesiredState(ctx context.Context, req *NodeId) (*DesiredSpec, error) {
	return s.specFor(req.Id)
}

func (s *Server) SetDesiredState(ctx context.Context, req *SetDesiredStateRequest) (*Empty, error) {
	body, err := SpecToYAML(req.Spec.Engines)
	if err != nil {
		return nil, err
	}
	if err := s.st.WriteSpec(req.Node, body); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

// Watch pushes the current spec immediately, then every subsequent change to
// that node's slice. The stream is the slave's only channel to the master.
func (s *Server) Watch(req *WatchRequest, stream grpc.ServerStreamingServer[DesiredSpec]) error {
	ch := s.st.Subscribe(req.Node)
	defer s.st.Unsubscribe(req.Node, ch)
	send := func() error {
		spec, err := s.specFor(req.Node)
		if err != nil {
			return err
		}
		return stream.Send(spec)
	}
	if err := send(); err != nil {
		return err
	}
	for {
		select {
		case <-ch:
			if err := send(); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}

func (s *Server) ReportStatus(ctx context.Context, st *Status) (*Empty, error) {
	body := st.State
	if st.Detail != "" {
		body = st.State + ": " + st.Detail
	}
	if err := s.st.ReportStatus(st.Node, body); err != nil {
		return nil, err
	}
	return &Empty{}, nil
}

func (s *Server) ListStatus(ctx context.Context, _ *Empty) (*StatusList, error) {
	rows, err := s.st.ListStatus()
	if err != nil {
		return nil, err
	}
	out := make([]*Status, 0, len(rows))
	for _, r := range rows {
		state, detail, _ := strings.Cut(r.Body, ": ")
		if detail == "" {
			state = r.Body
		}
		out = append(out, &Status{Node: r.Node, State: state, Detail: detail, Ts: r.TS})
	}
	return &StatusList{Statuses: out}, nil
}

func (s *Server) specFor(node string) (*DesiredSpec, error) {
	body, ok, err := s.st.GetSpec(node)
	if err != nil {
		return nil, fmt.Errorf("get spec: %w", err)
	}
	if !ok {
		return &DesiredSpec{}, nil
	}
	engines, err := SpecFromYAML(body)
	if err != nil {
		return nil, err
	}
	return &DesiredSpec{Engines: engines}, nil
}