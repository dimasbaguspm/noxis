// Package uat is the acceptance suite: the real compose stack (noxis + a
// target postgres) comes up and is driven exclusively through its public
// REST surface — the same moves an operator makes, encoded in Gherkin.
// Run via `make uat` (godog).
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
// dir as cwd, so stack paths resolve against it.
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
	baseURL   = "http://localhost:9000"

	connHost = "db"         // the compose postgres target
	connPass = "noxis"      // its password (dev stack only)
	tickKey  = "noxis-tick" // NOXIS_TICK_KEY in the compose stack
)

// world carries state between steps; godog runs scenarios sequentially and
// hands each one a fresh world.
type world struct {
	code     int
	conn     string // connection name under test
	changeID string
	key      redisKey // the redis key a scenario picked out of a browse
	page     redisPage
	result   resultBody
	change   changeBody
	body     string // the last response body, for error assertions

	database       mongoDatabase   // the mongo database a scenario picked out
	collection     mongoCollection // the mongo collection a scenario picked out
	mongoQueueSize int
	ticked         tickBody // the last /internal/tick result
}

// tickBody is the /internal/tick response shape.
type tickBody struct {
	Applied []string `json:"applied"`
	Failed  []string `json:"failed"`
}


// redisSeed runs the CLI inside a compose key-value service: the scenarios
// need keys to browse, and noxis has no endpoint that writes one. The service
// and its CLI differ per server (redis-cli / valkey-cli).
func redisSeed(service string, args ...string) error {
	cli := "redis-cli"
	if service == "valkey" {
		cli = "valkey-cli"
	}
	full := append([]string{"compose", "-f", stackFile, "exec", "-T", service, cli}, args...)
	out, err := exec.Command("docker", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", cli, strings.Join(args, " "), err, out)
	}
	return nil
}

type resultBody struct {
	Columns  []string   `json:"columns"`
	Rows     [][]string `json:"rows"`
	Affected int64      `json:"affected"`
	Message  string     `json:"message"`
}

type checkBody struct {
	Target string `json:"target"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
	Error  string `json:"error"`
}

func stackUp() error {
	out, err := exec.Command("docker", "compose", "-f", stackFile, "up", "-d", "--build", "--remove-orphans").CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose up: %w\n%s", err, out)
	}
	return waitHTTP(baseURL+"/healthz", 10*time.Second)
}

func stackDown() {
	_ = exec.Command("docker", "compose", "-f", stackFile, "down", "-v", "--remove-orphans").Run()
}

func waitHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("%s not ready within %v", url, timeout)
}

// api hits the noxis REST surface with an optional JSON body.
func (w *world) api(method, path string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	w.code = resp.StatusCode
	w.body = strings.TrimSpace(string(b))
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("decode %s %s: %w (%s)", method, path, err, strings.TrimSpace(string(b)))
		}
	}
	return nil
}

// raw returns the response body of a GET (for presence assertions).
func (w *world) raw(path string) (code int, body string, err error) {
	resp, err := http.Get(baseURL + path)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	return resp.StatusCode, string(b), nil
}
