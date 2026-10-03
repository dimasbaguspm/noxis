// Package uat is the acceptance suite: the real compose stack (Kafka +
// master + one slave) comes up on host docker and is driven exclusively
// through its public REST surface — the same moves an operator makes,
// encoded in Gherkin feature files. Run via `make uat` (godog); default run
// skips @slow scenarios (image pulls), UAT_TAGS overrides.
//
// The steps deliberately never touch the internal wire client, kafka, or the
// docker repo — the contract under test is what a user can do, not how it is
// implemented.
package uat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// root is the module root, found via go.mod: `go test` runs with the package
// dir as cwd, so all stack paths resolve against it.
var root = func() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("getwd: %v", err))
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found above " + dir)
		}
		dir = parent
	}
}()

var (
	stackFile = root + "/docker/docker-compose.yml"
	masterURL = "http://localhost:9000"
	dbFile    = root + "/docker/db.sqlite"

	// engineLabel marks slave-created containers; compose down won't see
	// them, so stackDown sweeps by label/prefix (see stackDown). nodeName is
	// what the compose slave registers as (NOXIS_SLAVE_NAME).
	engineLabel = "noxis.engine"
	nodeName    = "zeus"

	waitHealth   = 2 * time.Minute
	waitRegister = 2 * time.Minute
	waitConverge = 2 * time.Minute
	waitOp       = 45 * time.Second
)

// world is the state a scenario carries between steps. godog runs scenarios
// sequentially and hands each one a fresh world, so no locking is needed.
type world struct {
	code int    // last doorman-style op status
	out  string // last doorman-style op output
}

// stackUp seeds the registry file (an empty sqlite file is valid; the master
// creates the schema), starts the compose stack, and waits for the master to
// answer /healthz.
func stackUp() error {
	if _, err := os.Stat(dbFile); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat %s: %w", dbFile, err)
		}
		if err := os.WriteFile(dbFile, nil, 0o644); err != nil {
			return fmt.Errorf("seed %s: %w", dbFile, err)
		}
	}
	out, err := exec.Command("docker", "compose", "-f", stackFile, "up", "-d", "--build").CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose up: %w\n%s", err, out)
	}
	return waitHTTP(masterURL+"/healthz", http.StatusOK, waitHealth)
}

// stackDown stops the stack and sweeps host-side artifacts the slave created
// outside compose's scope (containers, volumes, the node network).
func stackDown() {
	_ = exec.Command("docker", "compose", "-f", stackFile, "down", "-v").Run()
	_ = sh("docker rm -f $(docker ps -aq --filter label=%s)", engineLabel)
	_ = sh("docker volume rm -f $(docker volume ls -q --filter name=noxis-%s)", nodeName)
	_ = sh("docker network rm %s", "noxis-"+nodeName)
}

// sh runs a shell snippet (needed for the $(...) expansion filters).
func sh(format string, args ...any) error {
	out, err := exec.Command("sh", "-c", fmt.Sprintf(format, args...)).CombinedOutput()
	if err != nil && len(out) > 0 {
		return fmt.Errorf("sh: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return err
}

func waitHTTP(url string, want int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == want {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("%s not %d within %v", url, want, timeout)
}

// api hits the master's REST surface with an optional JSON body.
func (w *world) api(method, path string, body any) (code int, b []byte, err error) {
	var rd io.Reader
	if body != nil {
		b, err = json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, masterURL+path, rd)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, b, nil
}

type statusNode struct {
	Node   string `json:"node"`
	State  string `json:"state"`
	Detail string `json:"detail"`
}

func (w *world) statuses() ([]statusNode, error) {
	code, b, err := w.api(http.MethodGet, "/v1/status", nil)
	if err != nil {
		return nil, err
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("status: %d %s", code, b)
	}
	var body struct {
		Nodes []statusNode `json:"nodes"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	return body.Nodes, nil
}

// pollOp posts a doorman-style op and retries while the engine looks not
// ready (container still created, postgres mid-boot, image pull). Guard
// verdicts (400) and real engine errors come back as-is.
func (w *world) pollOp(path string, body map[string]string) (code int, out string, err error) {
	deadline := time.Now().Add(waitOp)
	var b []byte
	for {
		code, b, err = w.api(http.MethodPost, path, body)
		if err != nil {
			return 0, "", err
		}
		out = opOut(b)
		if !retryable(code, out) {
			return code, out, nil
		}
		if time.Now().After(deadline) {
			return code, out, fmt.Errorf("%s: still not ready after %v (last %d %q)", path, waitOp, code, out)
		}
		time.Sleep(3 * time.Second)
	}
}

func retryable(code int, out string) bool {
	if code != http.StatusInternalServerError {
		return false
	}
	phrases := []string{"no such container", "not found", "connection refused",
		"could not connect", "starting up", "Can't connect"}
	for _, p := range phrases {
		if strings.Contains(out, p) {
			return true
		}
	}
	return false
}

func opOut(b []byte) string {
	var body struct {
		Output string `json:"output"`
	}
	if err := json.Unmarshal(b, &body); err != nil {
		return strings.TrimSpace(string(b))
	}
	return strings.TrimSpace(body.Output)
}

// ---- shared step helpers ----

func (w *world) waitState(name, state string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		nodes, err := w.statuses()
		if err == nil {
			for _, n := range nodes {
				if n.Node == name && n.State == state {
					return nil
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("node %s not %q within %v", name, state, timeout)
}

func (w *world) waitDetail(name, needle string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		nodes, err := w.statuses()
		if err == nil {
			for _, n := range nodes {
				if n.Node == name && strings.Contains(n.Detail, needle) {
					return nil
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("node %s detail missing %q within %v", name, needle, timeout)
}

func (w *world) waitContainer(name, image string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "inspect", "-f", "{{.Config.Image}}", name).CombinedOutput()
		if err == nil && strings.Contains(strings.TrimSpace(string(out)), image) {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("container %s not on image %q within %v", name, image, timeout)
}

// doorman runs one op and remembers the verdict for later assertions.
func (w *world) doorman(op, engine, sql string) error {
	path := "/v1/nodes/" + nodeName + "/doorman/" + op
	code, out, err := w.pollOp(path, map[string]string{"engine": engine, "sql": sql})
	if err != nil {
		return err
	}
	w.code, w.out = code, out
	return nil
}
