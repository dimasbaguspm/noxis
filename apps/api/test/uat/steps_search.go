package uat

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/cucumber/godog"
)

// Search-platform scenarios browse what a cluster indexes, query it, and review a delete. curl inside the container
// seeds, because noxis has no endpoint that writes an index.

// searchResult is a query answer: the documents the platform returned, as JSON.
type searchResult struct {
	Documents []string `json:"documents"`
}

func (w *world) searchSteps(sc *godog.ScenarioContext) {
	sc.Step(`^an elasticsearch connection named "([^"]*)"$`, w.registerSearch)
	sc.Step(`^I seed elasticsearch document "([^"]*)" in "([^"]*)" with (.+)$`, w.searchSeed)
	sc.Step(`^the elasticsearch targets hold "([^"]*)"$`, w.searchTargets)
	sc.Step(`^I search elasticsearch index "([^"]*)" for (.+)$`, w.searchRun)
	sc.Step(`^the search result has (\d+) document\(s\)$`, w.searchDocumentCount)
	sc.Step(`^I request an elasticsearch "([^"]*)" on "([^"]*)" with query (.+)$`, w.searchRequestChange)
	sc.Step(`^the search request reports an impact of (\d+)$`, w.searchImpact)
}

// registerSearch registers the compose elasticsearch service, which runs without security on a single node.
func (w *world) registerSearch(ctx context.Context, name string) error {
	t := targets["elasticsearch"]
	conn := map[string]any{
		"name": name, "driver": t.Driver, "host": t.Host, "port": t.Port, "database": t.Database,
	}
	if err := w.api(http.MethodPost, enginePathOf(t.Driver, "/connections"), conn, nil); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("register elasticsearch connection: %d", w.code)
	}
	w.conn, w.host, w.targetKey = name, t.Host, "elasticsearch"
	return nil
}

// searchSeed writes one document straight into an index, then refreshes so the query sees it.
func (w *world) searchSeed(ctx context.Context, id, index, body string) error {
	// The cluster is the slowest service in the stack: a cold JVM answers long after noxis does, so the seed waits for it.
	if err := awaitTarget(90*time.Second, func() error {
		return searchCurl("http://localhost:9200/_cluster/health?wait_for_status=yellow&timeout=5s")
	}); err != nil {
		return err
	}
	base := "http://localhost:9200/" + index
	if err := searchCurl("-X", "PUT", "-H", "Content-Type: application/json", "-d", body, base+"/_doc/"+id); err != nil {
		return err
	}
	return searchCurl("-X", "POST", base+"/_refresh")
}

// searchCurl runs curl inside the elasticsearch container.
func searchCurl(args ...string) error {
	full := append([]string{"compose", "-f", stackFile, "--profile", "all", "exec", "-T", "elasticsearch", "curl", "-sS"}, args...)
	out, err := exec.Command("docker", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("elasticsearch %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return nil
}

// searchTargets proves the console's target list reaches the cluster.
func (w *world) searchTargets(ctx context.Context, want string) error {
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

// searchRun answers one query through the console's search route.
func (w *world) searchRun(ctx context.Context, index, query string) error {
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

func (w *world) searchDocumentCount(ctx context.Context, want int) error {
	if len(w.search.Documents) != want {
		return fmt.Errorf("search returned %d document(s), want %d", len(w.search.Documents), want)
	}
	return nil
}

// searchRequestChange submits a delete for review, the way the console does.
func (w *world) searchRequestChange(ctx context.Context, op, index, query string) error {
	w.change = changeBody{}
	body := map[string]any{
		"connection": w.conn, "notes": "uat " + op, "op": op, "target": index, "query": query,
	}
	if err := w.api(http.MethodPost, w.enginePath("/changes"), body, &w.change); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("submit search request: %d", w.code)
	}
	return nil
}

// searchImpact proves the dry run reported how many documents the delete would remove.
func (w *world) searchImpact(ctx context.Context, want int) error {
	if w.change.Status != "validated" {
		return fmt.Errorf("request status %q want validated (checks: %+v)", w.change.Status, w.change.Checks)
	}
	if w.change.EstAffected != int64(want) {
		return fmt.Errorf("impact %d want %d (checks: %+v)", w.change.EstAffected, want, w.change.Checks)
	}
	return nil
}
