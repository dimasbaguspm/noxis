package kit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Stub records live responses and replays them, keyed by "<METHOD> <path>". A scenario records with Record set and replays without it, so one scenario runs against the real stack or fully offline.
type Stub struct {
	Dir    string
	Record bool
}

// envelope is the on-disk form of one response.
type envelope struct {
	Status  int         `json:"status"`
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

// keyClean squeezes a request key into a filesystem-safe name.
var keyClean = regexp.MustCompile(`[^A-Za-z0-9]+`)

// file is the store path for a request.
func (s *Stub) file(r Request) string {
	name := keyClean.ReplaceAllString(strings.TrimSpace(r.Method+" "+r.Path), "_")
	return filepath.Join(s.Dir, name+".json")
}

// record writes a live response to the store.
func (s *Stub) record(r Request, resp Response) error {
	env := envelope{Status: resp.Status, Headers: resp.Headers, Body: string(resp.Body)}
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.file(r), b, 0o644)
}

// replay reads a stored response. A missing entry names the fix.
func (s *Stub) replay(r Request) (Response, error) {
	b, err := os.ReadFile(s.file(r))
	if err != nil {
		return Response{}, fmt.Errorf("no stub for %s %s (record it with UAT_RECORD=1): %w", r.Method, r.Path, err)
	}
	var env envelope
	if err := json.Unmarshal(b, &env); err != nil {
		return Response{}, fmt.Errorf("stub %s: %w", s.file(r), err)
	}
	return Response{Status: env.Status, Headers: env.Headers, Body: []byte(env.Body)}, nil
}
