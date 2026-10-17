// Package uat is the acceptance suite: the real compose stack (noxis + a
// target postgres) comes up and is driven exclusively through its public
// REST surface — the same moves an operator makes, encoded in Gherkin.
// Run via `make uat` (godog).
package uat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"noxis/test/uat/kit"
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
	stackFile   = root + "/docker-compose.yml"
	snapshotDir = root + "/apps/api/test/uat/snapshots"
	stubDir     = root + "/apps/api/test/uat/stubs"

	// Read from the compose file by loadTargets, so the suite and the stack share
	// one source of truth for the base URL and the credentials.
	baseURL         string
	adminUser       string
	adminPassword   string
	schedulerKey    string
	schedulerSecret string
	targets         map[string]target
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
	client    kit.Client
	snapshots kit.Snapshots
	headers   map[string]string // per-scenario header overrides
	cookies []*http.Cookie    // per-scenario cookies
	resp    kit.Response      // the last response
	code    int               // resp.Status, kept for the existing assertions
	body    string            // resp.String(), kept for the existing assertions

	conn      string // connection name under test
	host      string // the compose service backing that connection
	targetKey string // the key into targets for that connection
	changeID  string
	key      redisKey // the redis key a scenario picked out of a browse
	page     redisPage
	result   resultBody
	change   changeBody

	database       mongoDatabase   // the mongo database a scenario picked out
	collection     mongoCollection // the mongo collection a scenario picked out
	mongoQueueSize int
	ticked         tickBody // the last /internal/callback/scheduler result

	snaps map[string]dbSnapshot // named database dumps, for this scenario
}

// reset clears the per-scenario state. godog reuses one world across scenarios,
// so this runs before each one and gives every scenario a clean client.
func (w *world) reset() {
	jar, _ := cookiejar.New(nil)
	*w = world{
		client:    kit.Client{BaseURL: baseURL, Token: token, Jar: jar},
		snapshots: kit.Snapshots{Dir: snapshotDir, Update: envTrue("UPDATE_SNAPSHOTS")},
		headers:   map[string]string{},
	}
}

// send is the suite's single request path: it folds in the per-scenario headers
// and cookies and captures the response whole.
func (w *world) send(r kit.Request) error {
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	for k, v := range w.headers {
		r.Headers[k] = v
	}
	r.Cookies = append(r.Cookies, w.cookies...)
	resp, err := w.client.Send(r)
	if err != nil {
		return err
	}
	w.resp = resp
	w.code = resp.Status
	w.body = resp.String()
	return nil
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

// stackUp brings the compose stack up. It rebuilds only when UAT_BUILD=1; a run
// against a cold cache still builds, because compose builds a missing image.
// UAT_KEEP reuses the stack: the image and noxis's volume stay, but every
// container is recreated, so the target databases start empty.
func stackUp() error {
	if err := loadTargets(); err != nil {
		return err
	}
	if envTrue("UAT_KEEP") {
		_ = exec.Command("docker", "compose", "-f", stackFile, "rm", "-sf").Run()
	}
	args := []string{"compose", "-f", stackFile, "up", "-d", "--remove-orphans"}
	if envTrue("UAT_BUILD") {
		args = append(args, "--build")
	}
	out, err := exec.Command("docker", args...).CombinedOutput()
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
	c := kit.Client{BaseURL: baseURL}
	resp, err := c.Send(kit.Request{
		Method: http.MethodPost,
		Path:   "/external/v1/auth/login",
		Body:   []byte(`{"username":"` + adminUser + `","password":"` + adminPassword + `"}`),
	})
	if err != nil {
		return err
	}
	if resp.Status != http.StatusOK {
		return fmt.Errorf("login: status %d body %s", resp.Status, resp.String())
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := resp.Decode(&out); err != nil {
		return fmt.Errorf("login decode: %w", err)
	}
	token = out.Token
	return nil
}

// stackDown tears the stack down, volumes included, unless UAT_KEEP=1 leaves it
// up so the next run reuses it.
func stackDown() {
	if envTrue("UAT_KEEP") {
		return
	}
	_ = exec.Command("docker", "compose", "-f", stackFile, "down", "-v", "--remove-orphans").Run()
}

// envTrue reports whether an env switch is set to a truthy value.
func envTrue(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes":
		return true
	}
	return false
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

// api hits the noxis REST surface with an optional JSON body and decodes the
// reply into out.
func (w *world) api(method, path string, body any, out any) error {
	var raw []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		raw = b
	}
	if err := w.send(kit.Request{Method: method, Path: path, Body: raw}); err != nil {
		return err
	}
	if out != nil {
		if err := w.resp.Decode(out); err != nil {
			return fmt.Errorf("decode %s %s: %w (%s)", method, path, err, w.resp.String())
		}
	}
	return nil
}

// raw returns the response of a GET (for presence assertions).
func (w *world) raw(path string) (code int, body string, err error) {
	if err := w.send(kit.Request{Method: http.MethodGet, Path: path}); err != nil {
		return 0, "", err
	}
	return w.resp.Status, w.resp.String(), nil
}
