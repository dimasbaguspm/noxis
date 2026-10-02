package doorman

import (
	"context"
	"strings"
	"testing"

	"noxis/internal/entity"
)

// Fake docker driver: an in-memory container table, recording every command.
func fakeDocker(t *testing.T) {
	t.Helper()
	type box struct{ image, state string }
	boxes := map[string]box{}
	var cmds []string
	dockerCmd = func(_ context.Context, args ...string) (string, error) {
		cmds = append(cmds, strings.Join(args, " "))
		switch args[0] {
		case "ps":
			var out strings.Builder
			for name, b := range boxes {
				out.WriteString(name + "\t" + b.image + "\t" + b.state + "\n")
			}
			return out.String(), nil
		case "run":
			name := args[argsIndex(args, "--name")+1]
			image := args[len(args)-1]
			boxes[name] = box{image: image, state: "running"}
		case "rm":
			delete(boxes, args[2]) // docker rm -f NAME
		case "start":
			if b, ok := boxes[args[1]]; ok {
				b.state = "running"
				boxes[args[1]] = b
			}
		}
		return "", nil
	}
	t.Cleanup(func() {
		if len(boxes) > 0 {
			t.Errorf("leftover containers: %v", boxes)
		}
	})
	lookup := func(name string) box { return boxes[name] }
	_ = lookup
}

func argsIndex(args []string, want string) int {
	for i, a := range args {
		if a == want {
			return i
		}
	}
	return -1
}

func TestReconcile(t *testing.T) {
	fakeDocker(t)
	r := NewReconciler("zeus")
	ctx := context.Background()

	// 1. start two engines
	detail, err := r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "7"},
		{Name: "es", Type: "elastic", Version: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "started redis (redis:7)") || !strings.Contains(detail, "started es (elasticsearch:8)") {
		t.Fatalf("unexpected detail: %q", detail)
	}

	// 2. no-op when in sync
	std, err := r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "7"},
		{Name: "es", Type: "elastic", Version: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if std != "" {
		t.Fatalf("expected no-op, got %q", std)
	}

	// 3. version bump recreates
	detail, err = r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "8"},
		{Name: "es", Type: "elastic", Version: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "upgraded redis (redis:8)") {
		t.Fatalf("expected upgrade, got %q", detail)
	}

	// 4. removed engine cleans up
	detail, err = r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "removed noxis-zeus-es") {
		t.Fatalf("expected removal, got %q", detail)
	}

	// 5. unknown type is a clear error
	if _, err := r.Reconcile(ctx, []entity.Engine{{Name: "x", Type: "couchdb", Version: "4"}}); err == nil {
		t.Fatal("expected error for unknown engine type")
	}

	// 6. empty desired state removes everything
	if _, err := r.Reconcile(ctx, nil); err != nil {
		t.Fatal(err)
	}
}

func TestLookupAliases(t *testing.T) {
	for name, want := range map[string]string{"pg": "postgres", "es": "elastic", "mysql": "mysql"} {
		d, ok := Lookup(name)
		if !ok || d.Type != want {
			t.Errorf("Lookup(%q) = %v, %v; want %q", name, d, ok, want)
		}
	}
}