package schema

import (
	"fmt"
	"strings"

	"noxis/internal/entity"
)

// assignTableIDs gives every table a graph id: the bare name when unique,
// otherwise the schema-qualified one, normalised and never duplicated.
func assignTableIDs(tables []entity.Table) {
	count := make(map[string]int, len(tables))
	for i := range tables {
		count[tables[i].Name]++
	}
	seen := make(map[string]int, len(tables))
	for i := range tables {
		raw := tables[i].Name
		if count[raw] > 1 {
			raw = tables[i].Schema + "_" + raw
		}
		id := normalizeID(raw, "table")
		seen[id]++
		if n := seen[id]; n > 1 {
			id = fmt.Sprintf("%s_%d", id, n)
		}
		tables[i].ID = id
	}
}

// normalizeID turns a catalog name into a graph identifier without losing uniqueness.
func normalizeID(raw, fallback string) string {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	if out == "" {
		return fallback
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "t_" + out
	}
	return out
}
