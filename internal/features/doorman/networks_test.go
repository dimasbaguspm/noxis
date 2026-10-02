package doorman

import (
	"context"
	"strings"
	"testing"

	"noxis/internal/entity"
)

// Drives every Reconcile branch with a fake docker, asserting the exact
// create/rm/ls command stream.
func TestNetworkReconcile(t *testing.T) {
	var calls []string
	lsOut := []byte("")
	run := func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) >= 2 && args[1] == "ls" {
			return lsOut, nil
		}
		return nil, nil
	}
	m := &NetworkManager{name: "noxis-zeus", run: run}
	one := []entity.Engine{{Name: "postgres", Type: "postgres"}}

	if d, err := m.Reconcile(context.Background(), one); err != nil || d != "network noxis-zeus created (1 engine(s))" {
		t.Fatalf("create: got %q, %v", d, err)
	}
	if last := calls[len(calls)-1]; !strings.Contains(last, "network create") || !strings.Contains(last, "noxis-zeus") {
		t.Fatalf("create call: %s", last)
	}

	lsOut = []byte("noxis-zeus\n")
	before := len(calls)
	if d, err := m.Reconcile(context.Background(), one); err != nil || d != "in sync" {
		t.Fatalf("in sync: got %q, %v", d, err)
	}
	if got := len(calls); got != before+1 { // only the ls
		t.Fatalf("in-sync should issue a single ls, calls=%d (was %d)", got, before)
	}

	if d, err := m.Reconcile(context.Background(), []entity.Engine{}); err != nil || d != "network noxis-zeus removed (spec empty)" {
		t.Fatalf("remove: got %q, %v", d, err)
	}
	if last := calls[len(calls)-1]; !strings.Contains(last, "network rm") {
		t.Fatalf("rm call: %s", last)
	}

	lsOut = []byte("")
	before = len(calls)
	if d, err := m.Reconcile(context.Background(), []entity.Engine{}); err != nil || d != "" {
		t.Fatalf("no-op: got %q, %v", d, err)
	}
	if got := len(calls); got != before+1 { // only the ls
		t.Fatalf("no-op should issue a single ls, calls=%d (was %d)", got, before)
	}
}
