// Package uat runs the acceptance suite against a real process: the noxisd binary the
// user gets, on a port, holding the environment a deployment would set.
package uat

import (
	"context"
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
	revSecret   = "sk_uat_rev_0123456789abcdef0123456789rev"
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
		if err := enginesUp(); err != nil {
			return 0, err
		}
		defer enginesDown()
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

// slow reports whether this run may talk to real engines. Without it the @slow scenarios are
// excluded rather than skipped silently: a scenario that never ran is not evidence.
func slow() bool { return os.Getenv("UAT_SLOW") == "1" }

const composeFile = "compose.yml"

// enginesUp boots the engines the @slow scenarios name and waits for them to answer. It removes any
// leftovers first: a container from an interrupted run holds the port and answers with a connection
// that dies mid-query, which reads like a product bug and is not one.
func enginesUp() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "down", "-v", "--remove-orphans").
		CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "uat: docker compose down (pre-run): %v\n%s\n", err, out)
	}
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "up", "-d", "--wait")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("docker compose up: %w\n%s", err, out)
	}
	// A healthy healthcheck means the server answers ping; it does not mean the account the steps
	// use can authenticate yet, and MySQL stays busy finishing its boot for a few seconds.
	return waitForLogin(ctx)
}

// waitForLogin logs in as the account the scenarios use, on both engines, until it works. Anything
// less than this let the first scenario fail with a connection the driver called invalid.
func waitForLogin(ctx context.Context) error {
	deadline := time.Now().Add(45 * time.Second)
	for {
		if err := loginAs("mysql", "mysql"); err != nil {
			if time.Now().After(deadline) {
				return err
			}
			time.Sleep(time.Second)
			continue
		}
		if err := loginAs("mariadb", "mariadb"); err != nil {
			if time.Now().After(deadline) {
				return err
			}
			time.Sleep(time.Second)
			continue
		}
		return nil
	}
}

func loginAs(service, client string) error {
	cmd := exec.Command("docker", "compose", "-f", composeFile, "exec", "-T", service,
		client, "-h127.0.0.1", "-unoxis", "-pnoxispw", "app", "-e", "SELECT 1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s is not ready for the steps' account: %w\n%s", service, err, out)
	}
	return nil
}

// enginesDown removes them, and their volumes, even when a scenario failed.
func enginesDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composeFile, "down", "-v")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "uat: docker compose down: %v\n%s\n", err, out)
	}
}

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
