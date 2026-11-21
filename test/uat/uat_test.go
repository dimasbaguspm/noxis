// Package uat runs the acceptance suite against a real process: the noxis binary the
// user gets, on a port, holding the environment a deployment would set.
package uat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"noxis/test/uat/steps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cucumber/godog"
)

const (
	agentSecret = "sk_uat_agent_0123456789abcdef0123456789"
	opsSecret   = "sk_uat_ops_0123456789abcdef0123456789aa"
	revSecret   = "sk_uat_rev_0123456789abcdef0123456789rev"
)

// TestMain builds noxis, starts `http serve` on a free port and hands the address to the steps
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

	binary := filepath.Join(dir, "noxis")
	build := exec.Command("go", "build", "-o", binary, "./cmd/noxis")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("build noxis: %w\n%s", err, out)
	}

	addr, err := freeAddr()
	if err != nil {
		return 0, err
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return 0, err
	}

	// The listen address and the state file are constants on purpose; the flags exist for this suite
	// and for development, so a second instance never needs an environment variable.
	cmd := exec.Command(binary, "http", "serve", "--addr", addr, "--state", filepath.Join(dir, "state.db"))
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"SECRET_KEY="+hex.EncodeToString(key),
		"TOKENS="+fmt.Sprintf(
			`{"agent":{"secret":%q,"scopes":["read","propose"]},"rev":{"secret":%q,"scopes":["read","propose","approve"]},"ops":{"secret":%q,"scopes":["read","propose","approve","apply","admin"]}}`,
			agentSecret, revSecret, opsSecret),
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
		"UAT_REV_SECRET":   revSecret,
	} {
		if err := os.Setenv(name, value); err != nil {
			return 0, err
		}
	}
	if slow() {
		addrs, stop, err := enginesUp(context.Background())
		if err != nil {
			return 0, err
		}
		defer stop()
		for engine, addr := range addrs {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return 0, err
			}
			for name, value := range map[string]string{
				"UAT_" + strings.ToUpper(engine) + "_HOST": host,
				"UAT_" + strings.ToUpper(engine) + "_PORT": port,
				"UAT_DB_USER":     dbUser,
				"UAT_DB_PASSWORD": dbPassword,
				"UAT_DB_NAME":     dbName,
			} {
				if err := os.Setenv(name, value); err != nil {
					return 0, err
				}
			}
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
		res, err := http.Get(base + "/healthz")
		if err == nil {
			_ = res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("noxis http serve did not answer %s/healthz within %s", base, within)
}

// slow reports whether this run may talk to real engines. Without it the @slow scenarios are
// excluded rather than skipped silently: a scenario that never ran is not evidence.
func slow() bool { return os.Getenv("UAT_SLOW") == "1" }

func TestAcceptance(t *testing.T) {
	tags := "~@slow"
	if slow() {
		tags = "@slow"
	}
	suite := godog.TestSuite{
		Name:                "noxis",
		ScenarioInitializer: steps.InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
			Tags:     tags,
			// Strict makes an undefined step a failure: a scenario that silently skips its
			// assertions is worse than no scenario at all.
			Strict: true,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("acceptance suite failed")
	}
}
