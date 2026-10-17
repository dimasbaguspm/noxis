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

// scratchRe matches the throwaway schema names a change verifies in, which
// differ on every run.
var scratchRe = regexp.MustCompile(`noxis_verify_[0-9a-zA-Z]+`)

// Snapshots compares response bodies against committed golden files. Update
// rewrites them instead of comparing; it is driven by UPDATE_SNAPSHOTS=1.
type Snapshots struct {
	Dir    string
	Update bool
}

// Match scrubs volatile fields from body and compares it against Dir/name.json.
// With Update set it writes that file instead. A missing golden is an error
// telling the reader how to create it.
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
		return fmt.Errorf("response does not match snapshot %s\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
	return nil
}

// scrubJSON rewrites a JSON body with its volatile fields replaced, indented so
// the golden diff reads well. HTML escaping is off, so a placeholder stays
// readable ("<id>", not "\u003cid\u003e").
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

// scrub walks the decoded body: name-keyed fields are replaced by a placeholder,
// and scratch schema names in any string are masked.
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

// placeholder reports the mask for a volatile key: ids, timestamps and
// durations differ on every run and must not enter a golden.
func placeholder(key string) (any, bool) {
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
