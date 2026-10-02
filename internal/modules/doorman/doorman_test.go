package doorman

import (
	"context"
	"strings"
	"testing"

	"noxis/internal/entity"
	"noxis/internal/repositories/docker"
)

// stubDocker records the ops the module decided on; the embedded nil client
// satisfies the untouched half of the seam.
type stubDocker struct {
	*docker.Client
	imageExists bool
	volExists   bool
	inspect     *docker.Container // nil → ErrNotFound (drives create/update)
	calls       []string
}

func (s *stubDocker) ContainerInspect(ctx context.Context, name string) (docker.Container, error) {
	s.calls = append(s.calls, "inspect "+name)
	if s.inspect == nil {
		return docker.Container{}, docker.ErrNotFound
	}
	return *s.inspect, nil
}
func (s *stubDocker) ContainerStop(ctx context.Context, name string) error {
	s.calls = append(s.calls, "stop "+name)
	return nil
}
func (s *stubDocker) ContainerRemove(ctx context.Context, name string) error {
	s.calls = append(s.calls, "rmc "+name)
	return nil
}
func (s *stubDocker) ImageExists(ctx context.Context, image string) (bool, error) {
	s.calls = append(s.calls, "image-exists "+image)
	return s.imageExists, nil
}
func (s *stubDocker) ImagePull(ctx context.Context, image string) error {
	s.calls = append(s.calls, "pull "+image)
	return nil
}
func (s *stubDocker) VolumeExists(ctx context.Context, name string) (bool, error) {
	s.calls = append(s.calls, "vol-exists "+name)
	return s.volExists, nil
}
func (s *stubDocker) VolumeCreate(ctx context.Context, name string) error {
	s.calls = append(s.calls, "vol-create "+name)
	return nil
}
func (s *stubDocker) ContainerRun(ctx context.Context, o docker.ContainerOptions) error {
	s.calls = append(s.calls, "run "+o.Name+" "+o.Image+" net="+o.Network)
	return nil
}
func (s *stubDocker) ContainerExec(ctx context.Context, name string, args ...string) (string, error) {
	s.calls = append(s.calls, "exec "+name+" "+strings.Join(args, " "))
	return "1\n", nil
}

func TestCreateInstance(t *testing.T) {
	stub := &stubDocker{}
	u := New(stub)
	eng := entity.Engine{Name: "postgres", Type: "postgres", Version: "16-alpine"}

	d, err := u.CreateInstance(context.Background(), "zeus", "noxis-zeus", eng)
	if err != nil {
		t.Fatal(err)
	}
	if d != "instance noxis-zeus-postgres created (postgres:16-alpine)" {
		t.Fatalf("detail: %s", d)
	}
	if last := stub.calls[len(stub.calls)-1]; last != "run noxis-zeus-postgres postgres:16-alpine net=noxis-zeus" {
		t.Fatalf("run: %s", last)
	}
	all := strings.Join(stub.calls, "\n")
	for _, want := range []string{"pull postgres:16-alpine", "vol-create noxis-zeus-postgres-16-data"} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q in:\n%s", want, all)
		}
	}
	if stub.calls[0] != "inspect noxis-zeus-postgres" {
		t.Fatalf("first op should be inspect: %v", stub.calls[0])
	}
}

func TestUpdateInstance(t *testing.T) {
	stub := &stubDocker{inspect: &docker.Container{Config: docker.ContainerConfig{Image: "postgres:15"}}}
	u := New(stub)
	eng := entity.Engine{Name: "postgres", Type: "postgres", Version: "16-alpine"}

	d, err := u.CreateInstance(context.Background(), "zeus", "noxis-zeus", eng)
	if err != nil {
		t.Fatal(err)
	}
	if d != "instance noxis-zeus-postgres updated (postgres:16-alpine)" {
		t.Fatalf("detail: %s", d)
	}
	all := strings.Join(stub.calls, "\n")
	for _, want := range []string{
		"stop noxis-zeus-postgres",
		"rmc noxis-zeus-postgres",
		"run noxis-zeus-postgres postgres:16-alpine net=noxis-zeus",
		"vol-create noxis-zeus-postgres-16-data", // major-split volume, 15 -> 16
	} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q in:\n%s", want, all)
		}
	}
}

func TestVerifyKinds(t *testing.T) {
	u := New(&stubDocker{})
	eng := entity.Engine{Name: "postgres", Type: "postgres"}

	if _, err := u.Query(context.Background(), "zeus", eng, "DROP TABLE t"); err == nil ||
		!strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("query must reject DDL, got %v", err)
	}
	if _, err := u.Query(context.Background(), "zeus", eng, "INSERT INTO t VALUES (1)"); err != nil {
		t.Fatalf("query must allow DML, got %v", err)
	}
	if _, err := u.AlterTable(context.Background(), "zeus", eng, "SELECT 1"); err == nil ||
		!strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("alter must reject reads, got %v", err)
	}
	if _, err := u.AlterTable(context.Background(), "zeus", eng, "ALTER TABLE t ADD COLUMN c int"); err != nil {
		t.Fatalf("alter must allow DDL, got %v", err)
	}
}

func TestRollbackWrapsPostgres(t *testing.T) {
	stub := &stubDocker{}
	u := New(stub)
	eng := entity.Engine{Name: "postgres", Type: "postgres"}

	if _, err := u.RollbackTable(context.Background(), "zeus", eng, "ALTER TABLE t DROP COLUMN c"); err != nil {
		t.Fatal(err)
	}
	got := stub.calls[len(stub.calls)-1]
	for _, want := range []string{"BEGIN;", "ROLLBACK;", "ALTER TABLE t DROP COLUMN c"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rollback argv missing %q: %s", want, got)
		}
	}
}

func TestUnsupportedEngine(t *testing.T) {
	u := New(&stubDocker{})
	eng := entity.Engine{Name: "x", Type: "mongodb"}
	if _, err := u.CreateInstance(context.Background(), "zeus", "n", eng); err == nil ||
		!strings.Contains(err.Error(), "sql only") {
		t.Fatalf("want unsupported-engine error, got %v", err)
	}
	if _, err := u.Query(context.Background(), "zeus", eng, "SELECT 1"); err == nil ||
		!strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("query on unsupported engine should error, got %v", err)
	}
}
