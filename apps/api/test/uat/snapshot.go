package uat

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
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

// snapshotters maps a driver to its implementation. A family without one is reported when a scenario asks for a snapshot; add its dump/restore here.
var snapshotters = map[string]snapshotter{
	"postgres": postgresSnapshot{},
	"mysql":    mysqlSnapshot{},
	"mariadb":  mysqlSnapshot{},
	"mongo":    mongoSnapshot{},
	"redis":    redisSnapshot{},
	"valkey":   redisSnapshot{},
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
	full := []string{"compose", "-f", stackFile, "--profile", "all", "exec", "-T"}
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
	full := []string{"compose", "-f", stackFile, "--profile", "all", "exec", "-T"}
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

// postgresSnapshot uses pg_dump / psql. --clean --if-exists makes the restore drop what it is about to recreate.
type postgresSnapshot struct{}

func (postgresSnapshot) dump(t target) ([]byte, error) {
	return execTarget(t, []string{"PGPASSWORD=" + t.Password},
		"pg_dump", "-U", t.User, "-d", t.Database, "--clean", "--if-exists")
}

func (postgresSnapshot) load(t target, data []byte) error {
	return execTargetIn(data, t, []string{"PGPASSWORD=" + t.Password},
		"psql", "-U", t.User, "-d", t.Database, "-q", "-v", "ON_ERROR_STOP=1")
}

// mysqlSnapshot uses the server's own dump/client pair. MySQL 8 and MariaDB ship different binary names, so the service picks them.
type mysqlSnapshot struct{}

func mysqlTools(t target) (dump, cli string) {
	if strings.Contains(t.Host, "mariadb") {
		return "mariadb-dump", "mariadb"
	}
	return "mysqldump", "mysql"
}

func (mysqlSnapshot) dump(t target) ([]byte, error) {
	dump, _ := mysqlTools(t)
	return execTarget(t, []string{"MYSQL_PWD=" + t.Password},
		dump, "-u"+t.User, "--databases", t.Database)
}

func (mysqlSnapshot) load(t target, data []byte) error {
	_, cli := mysqlTools(t)
	return execTargetIn(data, t, []string{"MYSQL_PWD=" + t.Password}, cli, "-u"+t.User)
}

// mongoSnapshot streams mongodump's archive out and mongorestore's back in, so nothing is left on the container filesystem.
type mongoSnapshot struct{}

func mongoURI(t target) string {
	return fmt.Sprintf("mongodb://%s:%s@localhost:%d/?authSource=admin", t.User, t.Password, t.Port)
}

func (mongoSnapshot) dump(t target) ([]byte, error) {
	return execTarget(t, nil, "mongodump", "--uri="+mongoURI(t), "--archive", "--gzip")
}

func (mongoSnapshot) load(t target, data []byte) error {
	return execTargetIn(data, t, nil, "mongorestore", "--uri="+mongoURI(t), "--archive", "--gzip", "--drop")
}

// redisSnapshot takes an RDB point-in-time dump and restores it by replacing the data file and restarting the server.
type redisSnapshot struct{}

func redisCLI(t target) string {
	if strings.Contains(t.Host, "valkey") {
		return "valkey-cli"
	}
	return "redis-cli"
}

const redisSnapshotFile = "/data/noxis-uat.rdb"

func (redisSnapshot) dump(t target) ([]byte, error) {
	cli := redisCLI(t)
	if _, err := execTarget(t, nil, cli, "--rdb", redisSnapshotFile); err != nil {
		return nil, err
	}
	return execTarget(t, nil, "cat", redisSnapshotFile)
}

func (redisSnapshot) load(t target, data []byte) error {
	// the server loads /data/dump.rdb at startup, so the dump is written there and the server restarted without saving over it
	if err := execTargetIn(data, t, nil, "sh", "-c", "cat > /data/dump.rdb"); err != nil {
		return err
	}
	id, err := containerID(t.Host)
	if err != nil {
		return err
	}
	// SIGKILL skips the shutdown save that would overwrite the restored file
	if _, err := exec.Command("docker", "kill", "-s", "KILL", id).CombinedOutput(); err != nil {
		return fmt.Errorf("kill %s: %w", t.Host, err)
	}
	if err := waitStopped(id, 10*time.Second); err != nil {
		return err
	}
	out, err := exec.Command("docker", "compose", "-f", stackFile, "--profile", "all", "start", t.Host).CombinedOutput()
	if err != nil {
		return fmt.Errorf("start %s: %w\n%s", t.Host, err, out)
	}
	cli := redisCLI(t)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if out, err := execTarget(t, nil, cli, "ping"); err == nil && strings.Contains(string(out), "PONG") {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("%s did not answer after restoring the snapshot", t.Host)
}

func containerID(service string) (string, error) {
	out, err := exec.Command("docker", "compose", "-f", stackFile, "--profile", "all", "ps", "-q", service).Output()
	if err != nil {
		return "", fmt.Errorf("container id for %s: %w", service, err)
	}
	id := strings.TrimSpace(string(out))
	if id == "" {
		return "", fmt.Errorf("container id for %s: not running", service)
	}
	return id, nil
}

func waitStopped(id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", id).Output()
		if err == nil && strings.TrimSpace(string(out)) == "false" {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("container %s did not stop", id)
}
