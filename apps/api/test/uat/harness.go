// Package uat is the acceptance suite: the real compose stack (noxis + a
// target postgres) comes up and is driven exclusively through its public
// REST surface — the same moves an operator makes, encoded in Gherkin.
// Run via `make uat` (godog).
package uat

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// root is the repository root, found via .git: `go test` runs with the package
// dir as cwd (apps/api/test/uat), so stack paths resolve against it.
var root = func() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("getwd: %v", err))
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic(".git not found above " + dir)
		}
		dir = parent
	}
}()

var (
	stackFile = root + "/docker-compose.yml"
	baseURL   = "http://localhost:9000"

	connHost = "db"    // the compose postgres target
	connPass = "noxis" // its password (dev stack only)

	// must match the noxis service env in docker-compose.yml
	schedulerKey    = "noxis-scheduler"
	schedulerSecret = "noxis-scheduler-secret"
	adminUser       = "admin"
	adminPassword   = "noxis-admin"
)

// token is the local session the suite obtains once in BeforeSuite and sends as
// a bearer token on every /v1 request.
var token string

// signScheduler authenticates the trigger the way the external service must:
// api key as bearer + HMAC over "<timestamp>.<body>". Kept independent of the
// server middleware, so the suite cross-checks the scheme rather than reusing it.
func signScheduler(req *http.Request, body []byte) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(schedulerSecret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	req.Header.Set("Authorization", "Bearer "+schedulerKey)
	req.Header.Set("X-Noxis-Timestamp", timestamp)
	req.Header.Set("X-Noxis-Signature", hex.EncodeToString(mac.Sum(nil)))
}

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
	ticked         tickBody // the last /internal/callback/scheduler result
}

// tickBody is the /internal/callback/scheduler response shape.
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
	if err := waitHTTP(baseURL+"/healthz", 10*time.Second); err != nil {
		return err
	}
	return login()
}

// login exchanges the dev root admin for a session token; every /v1 call needs
// it now that there is no open mode.
func login() error {
	body := strings.NewReader(`{"username":"` + adminUser + `","password":"` + adminPassword + `"}`)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login: status %d body %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("login decode: %w", err)
	}
	token = out.Token
	return nil
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
	req, err := http.NewRequest(http.MethodGet, baseURL+path, http.NoBody)
	if err != nil {
		return 0, "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
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
