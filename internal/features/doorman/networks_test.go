package doorman

import (
	"context"
	"testing"

	"noxis/internal/entity"
)

// stubNetworks records which docker ops the manager decided on.
type stubNetworks struct {
	exists bool
	calls  []string
}

func (s *stubNetworks) NetworkExists(ctx context.Context, label string) (bool, error) {
	s.calls = append(s.calls, "exists")
	return s.exists, nil
}
func (s *stubNetworks) NetworkCreate(ctx context.Context, name string, labels ...string) error {
	s.calls = append(s.calls, "create "+name)
	return nil
}
func (s *stubNetworks) NetworkRemove(ctx context.Context, name string) error {
	s.calls = append(s.calls, "rm "+name)
	return nil
}

// Drives every Reconcile branch: create, in-sync, rm, no-op.
func TestNetworkReconcile(t *testing.T) {
	m := &NetworkManager{name: "noxis-zeus", d: &stubNetworks{}}
	one := []entity.Engine{{Name: "postgres", Type: "postgres"}}

	if d, err := m.Reconcile(context.Background(), one); err != nil || d != "network noxis-zeus created (1 engine(s))" {
		t.Fatalf("create: got %q, %v", d, err)
	}
	if got := m.d.(*stubNetworks).calls; got[len(got)-1] != "create noxis-zeus" {
		t.Fatalf("create decision: %v", got)
	}

	m.d.(*stubNetworks).exists = true
	if d, err := m.Reconcile(context.Background(), one); err != nil || d != "in sync" {
		t.Fatalf("in sync: got %q, %v", d, err)
	}
	if got := m.d.(*stubNetworks).calls; got[len(got)-1] != "exists" {
		t.Fatalf("in-sync should only probe: %v", got)
	}

	if d, err := m.Reconcile(context.Background(), []entity.Engine{}); err != nil || d != "network noxis-zeus removed (spec empty)" {
		t.Fatalf("rm: got %q, %v", d, err)
	}
	if got := m.d.(*stubNetworks).calls; got[len(got)-1] != "rm noxis-zeus" {
		t.Fatalf("rm decision: %v", got)
	}

	m.d.(*stubNetworks).exists = false
	if d, err := m.Reconcile(context.Background(), []entity.Engine{}); err != nil || d != "" {
		t.Fatalf("no-op: got %q, %v", d, err)
	}
}
