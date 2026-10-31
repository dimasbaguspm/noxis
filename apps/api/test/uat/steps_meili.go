package uat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// Meilisearch scenarios browse indexes, search one, and review a delete. curl inside the container seeds, because noxis
// has no endpoint that writes documents, and the server applies both settings and documents as tasks.

func (w *world) meiliSteps(sc *godog.ScenarioContext) {
	sc.Step(`^a meilisearch connection named "([^"]*)"$`, w.registerMeili)
	sc.Step(`^I seed meilisearch index "([^"]*)" with (.+)$`, w.meiliSeed)
	sc.Step(`^the meilisearch targets hold "([^"]*)"$`, w.meiliTargets)
	sc.Step(`^I search meilisearch index "([^"]*)" for (.+)$`, w.meiliSearch)
	sc.Step(`^the meilisearch result has (\d+) document\(s\)$`, w.meiliCount)
	sc.Step(`^I request a meilisearch "([^"]*)" on "([^"]*)" with query (.+)$`, w.meiliRequestChange)
	sc.Step(`^the meilisearch request reports an impact of (\d+)$`, w.meiliImpact)
}

// registerMeili registers the compose meilisearch service; its master key travels as the connection's password.
func (w *world) registerMeili(ctx context.Context, name string) error {
	t := targets["meilisearch"]
	conn := map[string]any{
		"name": name, "driver": t.Driver, "host": t.Host, "port": t.Port,
		"password": t.Password, "database": t.Database,
	}
	if err := w.api(http.MethodPost, enginePathOf(t.Driver, "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register meilisearch connection: %d", w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, "meilisearch"
	return nil
}

// meiliSeed makes the index filterable and then loads the documents. Both are tasks the server runs in the background,
// so each is waited for: a query issued before the settings land would answer the wrong question.
func (w *world) meiliSeed(ctx context.Context, index, documents string) error {
	if err := awaitTarget(30*time.Second, func() error {
		_, err := meiliCurl("http://localhost:7700/health")
		return err
	}); err != nil {
		return err
	}
	base := "http://localhost:7700/indexes/" + index
	settings := `{"filterableAttributes":["user","role","team"]}`
	if err := w.meiliAwait("-X", "PATCH", "-H", "Content-Type: application/json", "-d", settings, base+"/settings"); err != nil {
		return err
	}
	return w.meiliAwait("-X", "PUT", "-H", "Content-Type: application/json", "-d", documents, base+"/documents")
}

// meiliAwait runs one request and waits for the task it accepted.
func (w *world) meiliAwait(args ...string) error {
	out, err := meiliCurl(args...)
	if err != nil {
		return err
	}
	var accepted struct {
		TaskUID int64 `json:"taskUid"`
	}
	if err := json.Unmarshal([]byte(out), &accepted); err != nil {
		return fmt.Errorf("meilisearch accepted no task: %w (%s)", err, out)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		task, err := meiliCurl("http://localhost:7700/tasks/" + strconv.FormatInt(accepted.TaskUID, 10))
		if err != nil {
			return err
		}
		switch {
		case strings.Contains(task, `"status":"succeeded"`):
			return nil
		case strings.Contains(task, `"status":"failed"`):
			return fmt.Errorf("meilisearch task %d failed: %s", accepted.TaskUID, task)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("meilisearch task %d did not finish: %s", accepted.TaskUID, task)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// meiliCurl runs curl inside the meilisearch container, which is where the master key already lives.
func meiliCurl(args ...string) (string, error) {
	full := append([]string{"compose", "-f", stackFile, "--profile", "all", "exec", "-T", "meilisearch",
		"curl", "-sS", "-H", "Authorization: Bearer noxis"}, args...)
	out, err := exec.Command("docker", full...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("meilisearch %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return string(out), nil
}

// meiliTargets proves the console's target list reaches the server.
func (w *world) meiliTargets(ctx context.Context, want string) error {
	var out struct {
		Targets []struct {
			Name string `json:"name"`
		} `json:"targets"`
	}
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn+"/targets"), nil, &out); err != nil {
		return err
	}
	for _, target := range out.Targets {
		if target.Name == want {
			return nil
		}
	}
	return fmt.Errorf("target list %+v lacks %q", out.Targets, want)
}

// meiliSearch answers one query through the console's search route.
func (w *world) meiliSearch(ctx context.Context, index, query string) error {
	body := map[string]any{"op": "search", "target": index, "query": query}
	var out searchResult
	if err := w.api(http.MethodPost, w.enginePath("/connections/"+w.conn+"/search"), body, &out); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("search: %d", w.code)
	}
	w.search = out
	return nil
}

func (w *world) meiliCount(ctx context.Context, want int) error {
	if len(w.search.Documents) != want {
		return fmt.Errorf("search returned %d document(s), want %d: %v", len(w.search.Documents), want, w.search.Documents)
	}
	return nil
}

// meiliRequestChange submits a delete for review, the way the console does.
func (w *world) meiliRequestChange(ctx context.Context, op, index, query string) error {
	w.change = changeBody{}
	body := map[string]any{
		"connection": w.conn, "notes": "uat " + op, "op": op, "target": index, "query": query,
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit meilisearch request: %d", w.code)
	}
	return nil
}

// meiliImpact proves the dry run reported how many documents the filter matches.
func (w *world) meiliImpact(ctx context.Context, want int) error {
	if w.change.Status != "validated" {
		return fmt.Errorf("request status %q want validated (checks: %+v)", w.change.Status, w.change.Checks)
	}
	if w.change.EstAffected != int64(want) {
		return fmt.Errorf("impact %d want %d (checks: %+v)", w.change.EstAffected, want, w.change.Checks)
	}
	return nil
}
