package kit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// scratchRe matches the throwaway schema names a change verifies in, which differ on every run.
var scratchRe = regexp.MustCompile(`noxis_verify_[0-9a-zA-Z]+`)

// Snapshots compares response bodies against committed golden files.
type Snapshots struct {
	Dir    string
	Update bool
}

// Match scrubs volatile fields from body and compares it against Dir/name.json.
func (s Snapshots) Match(name string, body []byte) error {
	got, err := scrubJSON(body)
	if err != nil {
		return fmt.Errorf("snapshot %s: %w", name, err)
	}
	path := filepath.Join(s.Dir, name+".json")
	if s.Update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, got, 0o644)
	}
	want, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("snapshot %s missing; rerun with UPDATE_SNAPSHOTS=1", name)
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		return fmt.Errorf("response does not match snapshot %s: %s\n--- want ---\n%s\n--- got ---\n%s", name, firstDifference(want, got), want, got)
	}
	return nil
}

// scrubJSON rewrites a JSON body with its volatile fields replaced, indented so the golden diff reads well.
func scrubJSON(body []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(scrub(v)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// scrub walks the decoded body: name-keyed fields are replaced by a placeholder, and scratch schema names in any string are masked.
func scrub(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if p, ok := placeholder(k); ok {
				t[k] = p
				continue
			}
			t[k] = scrub(val)
		}
		return t
	case []any:
		for i := range t {
			t[i] = scrub(t[i])
		}
		return t
	case string:
		return scratchRe.ReplaceAllString(t, "<scratch>")
	default:
		return v
	}
}

// firstDifference names the first line that differs, so a mismatch reads without scanning both bodies.
func firstDifference(want, got []byte) string {
	wantLines := strings.Split(string(bytes.TrimSpace(want)), "\n")
	gotLines := strings.Split(string(bytes.TrimSpace(got)), "\n")
	for i := 0; i < len(wantLines) && i < len(gotLines); i++ {
		if wantLines[i] != gotLines[i] {
			return fmt.Sprintf("line %d differs: want %q, got %q", i+1, wantLines[i], gotLines[i])
		}
	}
	switch {
	case len(wantLines) < len(gotLines):
		return fmt.Sprintf("golden ends at line %d; response has %d more line(s)", len(wantLines), len(gotLines)-len(wantLines))
	case len(gotLines) < len(wantLines):
		return fmt.Sprintf("response ends at line %d; golden has %d more line(s)", len(gotLines), len(wantLines)-len(gotLines))
	}
	return "bodies differ in trailing whitespace"
}

// placeholder reports the mask for a volatile or sensitive key: credentials never enter a golden, and ids, timestamps and durations differ on every run.
func placeholder(key string) (any, bool) {
	lower := strings.ToLower(key)
	for _, secret := range []string{"token", "password", "secret", "authorization", "cookie", "credential"} {
		if strings.Contains(lower, secret) {
			return "<secret>", true
		}
	}
	switch {
	case key == "id" || strings.HasSuffix(key, "Id"):
		return "<id>", true
	case strings.HasSuffix(key, "At"):
		return "<ts>", true
	case strings.HasSuffix(key, "Ms"):
		return 0, true
	}
	return nil, false
}
