// Package uat runs the acceptance suite against a real process: the noxisd binary the
// user gets, on a port, holding the environment a deployment would set.
package uat

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/cucumber/godog"

	"noxis/test/uat/steps"
)

const (
	agentSecret = "sk_uat_agent_0123456789abcdef0123456789"
	opsSecret   = "sk_uat_ops_0123456789abcdef0123456789aa"
)

// TestMain builds noxisd, starts it on a free port and hands the address to the steps
// through the environment, which is how the suite stays a black box.
func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uat:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	dir, err := os.MkdirTemp("", "noxis-uat-")
	if err != nil {
		return 0, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	binary := filepath.Join(dir, "noxisd")
	build := exec.Command("go", "build", "-o", binary, "./cmd/http")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("build noxisd: %w\n%s", err, out)
	}

	addr, err := freeAddr()
	if err != nil {
		return 0, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return 0, err
	}

	cmd := exec.Command(binary)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"HTTP_ADDR="+addr,
		"STATE="+filepath.Join(dir, "state.db"),
		"SECRET_KEY="+hex.EncodeToString(key),
		"LOG_LEVEL=warn",
		"TOKENS="+fmt.Sprintf(
			`{"agent":{"secret":%q,"scopes":["read","propose"]},"ops":{"secret":%q,"scopes":["read","propose","approve","apply","admin"]}}`,
			agentSecret, opsSecret),
	)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	base := "http://" + addr
	if err := waitForServer(base, 20*time.Second); err != nil {
		return 0, err
	}
	for name, value := range map[string]string{
		"UAT_URL":          base,
		"UAT_AGENT_SECRET": agentSecret,
		"UAT_OPS_SECRET":   opsSecret,
	} {
		if err := os.Setenv(name, value); err != nil {
			return 0, err
		}
	}

	return m.Run(), nil
}

func freeAddr() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String(), nil
}

func waitForServer(base string, within time.Duration) error {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		res, err := http.Get(base + "/healthz") //nolint:noctx // a boot probe outlives no request
		if err == nil {
			_ = res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("noxisd did not answer %s/healthz within %s", base, within)
}

func TestAcceptance(t *testing.T) {
	suite := godog.TestSuite{
		Name:                "noxis",
		ScenarioInitializer: steps.InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("acceptance suite failed")
	}
}
