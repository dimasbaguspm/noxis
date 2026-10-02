package doorman

import (
	"context"
	"testing"

	"noxis/internal/entity"
	"noxis/internal/repositories/docker"
)

// stubDocker records which ops the manager decided on; implements the whole
// interfaces.Docker seam so the decision layer stays testable offline.
type stubDocker struct {
	networkExists bool
	calls         []string
}

func (s *stubDocker) NetworkExists(ctx context.Context, label string) (bool, error) {
	s.calls = append(s.calls, "exists")
	return s.networkExists, nil
}
func (s *stubDocker) NetworkCreate(ctx context.Context, name string, labels ...string) error {
	s.calls = append(s.calls, "create "+name)
	return nil
}
func (s *stubDocker) NetworkRemove(ctx context.Context, name string) error {
	s.calls = append(s.calls, "rm "+name)
	return nil
}
func (s *stubDocker) ContainerList(ctx context.Context, label string) ([]string, error) {
	s.calls = append(s.calls, "list")
	return nil, nil
}
func (s *stubDocker) ContainerInspect(ctx context.Context, name string) (docker.Container, error) {
	s.calls = append(s.calls, "inspect "+name)
	return docker.Container{}, nil
}
func (s *stubDocker) ContainerRun(ctx context.Context, opts docker.ContainerOptions) error {
	s.calls = append(s.calls, "run "+opts.Name)
	return nil
}
func (s *stubDocker) ContainerRemove(ctx context.Context, name string) error {
	s.calls = append(s.calls, "rmc "+name)
	return nil
}

// Drives every Reconcile branch: create, in-sync, rm, no-op.
func TestNetworkReconcile(t *testing.T) {
	stub := &stubDocker{}
	m := &NetworkManager{name: "noxis-zeus", d: stub}
	one := []entity.Engine{{Name: "postgres", Type: "postgres"}}

	if d, err := m.Reconcile(context.Background(), one); err != nil || d != "network noxis-zeus created (1 engine(s))" {
		t.Fatalf("create: got %q, %v", d, err)
	}
	if got := stub.calls[len(stub.calls)-1]; got != "create noxis-zeus" {
		t.Fatalf("create decision: %v", got)
	}

	stub.networkExists = true
	if d, err := m.Reconcile(context.Background(), one); err != nil || d != "in sync" {
		t.Fatalf("in sync: got %q, %v", d, err)
	}
	if got := stub.calls[len(stub.calls)-1]; got != "exists" {
		t.Fatalf("in-sync should only probe: %v", got)
	}

	if d, err := m.Reconcile(context.Background(), []entity.Engine{}); err != nil || d != "network noxis-zeus removed (spec empty)" {
		t.Fatalf("rm: got %q, %v", d, err)
	}
	if got := stub.calls[len(stub.calls)-1]; got != "rm noxis-zeus" {
		t.Fatalf("rm decision: %v", got)
	}

	stub.networkExists = false
	if d, err := m.Reconcile(context.Background(), []entity.Engine{}); err != nil || d != "" {
		t.Fatalf("no-op: got %q, %v", d, err)
	}
}
