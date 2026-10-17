package uat

import (
	"bytes"
	"fmt"
	"os/exec"
)

// dbSnapshot is a target's dumped data, taken so a scenario can be restored.
type dbSnapshot struct {
	target target
	data   []byte
}

// snapshotter dumps and loads a target's data with its family's native tool.
type snapshotter interface {
	dump(t target) ([]byte, error)
	load(t target, data []byte) error
}

// snapshotters maps a driver to its implementation. A family without one is
// reported when a scenario asks for a snapshot; add its dump/restore here.
var snapshotters = map[string]snapshotter{
	"postgres": postgresSnapshot{},
	"mariadb":  mysqlSnapshot{},
}

func snapshotterFor(t target) (snapshotter, error) {
	s, ok := snapshotters[t.Driver]
	if !ok {
		return nil, fmt.Errorf("snapshot is not implemented for driver %q", t.Driver)
	}
	return s, nil
}

// execTarget runs a command in a target's container and returns its stdout.
func execTarget(t target, env []string, args ...string) ([]byte, error) {
	full := []string{"compose", "-f", stackFile, "exec", "-T"}
	for _, e := range env {
		full = append(full, "-e", e)
	}
	full = append(full, t.Host)
	full = append(full, args...)
	out, err := exec.Command("docker", full...).Output()
	if err != nil {
		return nil, fmt.Errorf("exec %s: %w", t.Host, err)
	}
	return out, nil
}

// execTargetIn pipes data into a command in a target's container.
func execTargetIn(data []byte, t target, env []string, args ...string) error {
	full := []string{"compose", "-f", stackFile, "exec", "-T"}
	for _, e := range env {
		full = append(full, "-e", e)
	}
	full = append(full, t.Host)
	full = append(full, args...)
	cmd := exec.Command("docker", full...)
	cmd.Stdin = bytes.NewReader(data)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("exec %s: %w\n%s", t.Host, err, out)
	}
	return nil
}

// postgresSnapshot uses pg_dump / psql. --clean --if-exists makes the restore
// drop what it is about to recreate.
type postgresSnapshot struct{}

func (postgresSnapshot) dump(t target) ([]byte, error) {
	return execTarget(t, []string{"PGPASSWORD=" + t.Password},
		"pg_dump", "-U", t.User, "-d", t.Database, "--clean", "--if-exists")
}

func (postgresSnapshot) load(t target, data []byte) error {
	return execTargetIn(data, t, []string{"PGPASSWORD=" + t.Password},
		"psql", "-U", t.User, "-d", t.Database, "-q", "-v", "ON_ERROR_STOP=1")
}

// mysqlSnapshot uses mysqldump / mysql. --databases emits the CREATE DATABASE
// and USE lines, so the restore is self-contained.
type mysqlSnapshot struct{}

func (mysqlSnapshot) dump(t target) ([]byte, error) {
	return execTarget(t, []string{"MYSQL_PWD=" + t.Password},
		"mariadb-dump", "-u"+t.User, "--databases", t.Database)
}

func (mysqlSnapshot) load(t target, data []byte) error {
	return execTargetIn(data, t, []string{"MYSQL_PWD=" + t.Password}, "mariadb", "-u"+t.User)
}
