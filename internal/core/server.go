package core

import (
	"context"
	"log/slog"

	"google.golang.org/grpc"

	"noxis/internal/interfaces"
)

// Server implements the Control service on the master. Stateless beyond the
// repositories: desired assignments (what should run) and the status echo
// (what nodes report running). Actual runtime never lives here.
type Server struct {
	UnimplementedControlServer
	specs  interfaces.SpecRepository
	status interfaces.StatusRepository
}

func NewServer(specs interfaces.SpecRepository, status interfaces.StatusRepository) *Server {
	return &Server{specs: specs, status: status}
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

// Watch pushes the current spec immediately, then every subsequent change to
// that node's slice. The stream is the slave's only channel to the master.
func (s *Server) Watch(req *WatchRequest, stream grpc.ServerStreamingServer[DesiredSpec]) error {
	ch := s.specs.Subscribe(req.Node)
	defer s.specs.Unsubscribe(req.Node, ch)
	send := func() error {
		spec, err := s.specs.Get(context.Background(), req.Node)
		if err != nil {
			return err
		}
		return stream.Send(SpecToProto(spec))
	}
	if err := send(); err != nil {
		return err
	}
	for {
		select {
		case <-ch:
			if err := send(); err != nil {
				slog.Warn("watch send failed", "node", req.Node, "error", err)
				return err
			}
		case <-stream.Context().Done():
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
