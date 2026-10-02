package doorman

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
)

// Fake docker driver: an in-memory container table recording every command.
func fakeDocker(t *testing.T) {
	t.Helper()
	type box struct{ id, state, desired string }
	boxes := map[string]box{} // container name -> box
	seq := 0
	runtimes.Exec = func(_ context.Context, args ...string) (string, error) {
		switch args[0] {
		case "ps": // docker ps -aq --filter ... -> ids
			ids := make([]string, 0, len(boxes))
			for _, b := range boxes {
				ids = append(ids, b.id)
			}
			return strings.Join(ids, "\n"), nil
		case "inspect": // docker inspect --format ... ID... -> name\tstate\tdesired
			var out strings.Builder
			for _, id := range args[argIndex(args, "--format")+2:] {
				for name, b := range boxes {
					if b.id == id {
						out.WriteString("/" + name + "\t" + b.state + "\t" + b.desired + "\n")
					}
				}
			}
			return out.String(), nil
		case "run":
			name := args[argIndex(args, "--name")+1]
			desired := ""
			for _, a := range args {
				if strings.HasPrefix(a, "noxis.desired=") {
					desired = strings.TrimPrefix(a, "noxis.desired=")
				}
			}
			seq++
			boxes[name] = box{id: fmt.Sprintf("c%d", seq), state: "running", desired: desired}
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
}

func argIndex(args []string, want string) int {
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

	// 1. start two engines with one custom port
	detail, err := r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "7", Ports: map[string]string{"6380": "6379"}},
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
		{Name: "redis", Type: "redis", Version: "7", Ports: map[string]string{"6380": "6379"}},
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
		{Name: "redis", Type: "redis", Version: "8", Ports: map[string]string{"6380": "6379"}},
		{Name: "es", Type: "elastic", Version: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "recreated redis (redis:8)") {
		t.Fatalf("expected recreate, got %q", detail)
	}

	// 4. port change recreates (config drift, same image)
	detail, err = r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "8", Ports: map[string]string{"6381": "6379"}},
		{Name: "es", Type: "elastic", Version: "8"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "recreated redis (redis:8)") {
		t.Fatalf("expected port-drift recreate, got %q", detail)
	}

	// 5. removed engine cleans up
	detail, err = r.Reconcile(ctx, []entity.Engine{
		{Name: "redis", Type: "redis", Version: "8", Ports: map[string]string{"6381": "6379"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail, "removed noxis-zeus-es") {
		t.Fatalf("expected removal, got %q", detail)
	}

	// 6. unknown type is a clear error
	if _, err := r.Reconcile(ctx, []entity.Engine{{Name: "x", Type: "couchdb", Version: "4"}}); err == nil {
		t.Fatal("expected error for unknown engine type")
	}

	// 7. empty desired state removes everything
	if _, err := r.Reconcile(ctx, nil); err != nil {
		t.Fatal(err)
	}
}

func TestLookupAliases(t *testing.T) {
	for name, want := range map[string]string{"pg": "postgres", "es": "elastic", "mysql": "mysql"} {
		d, ok := runtimes.Lookup(name)
		if !ok || d.Type != want {
			t.Errorf("Lookup(%q) = %v, %v; want %q", name, d, ok, want)
		}
	}
}
