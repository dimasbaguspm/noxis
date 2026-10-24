package schema

import (
	"fmt"
	"strings"

	"noxis/internal/entity"
)

// Mermaid renders tables as a Mermaid erDiagram. The diagram carries only what the catalog declares: a column is a key because a constraint says so, and a relationship exists because a foreign key says so.
func Mermaid(tables []entity.Table) string {
	if len(tables) == 0 {
		return "erDiagram\n"
	}
	ids := entityIDs(tables)
	var b strings.Builder
	b.WriteString("erDiagram\n")
	for i := range tables {
		fmt.Fprintf(&b, "    %s {\n", ids[i])
		for j := range tables[i].Columns {
			col := &tables[i].Columns[j]
			fmt.Fprintf(&b, "        %s %s%s\n", ident(col.Type, "value"), ident(col.Name, "column"), keySuffix(col))
		}
		b.WriteString("    }\n")
	}
	for i := range tables {
		for j := range tables[i].Columns {
			col := &tables[i].Columns[j]
			if col.References == nil {
				continue
			}
			// a foreign key to a table the connection does not list (another schema, a dropped table) cannot be drawn
			from := indexOf(tables, col.References)
			if from < 0 {
				continue
			}
			fmt.Fprintf(&b, "    %s ||--%s %s : %q\n", ids[from], cardinality(col), ids[i], col.Name)
		}
	}
	return b.String()
}

// entityIDs names every table: the bare name when it is unique, otherwise the schema-qualified one, always normalised and never duplicated.
func entityIDs(tables []entity.Table) []string {
	count := make(map[string]int, len(tables))
	for i := range tables {
		count[tables[i].Name]++
	}
	seen := make(map[string]int, len(tables))
	ids := make([]string, len(tables))
	for i := range tables {
		raw := tables[i].Name
		if count[raw] > 1 {
			raw = tables[i].Schema + "_" + raw
		}
		id := ident(raw, "table")
		seen[id]++
		if n := seen[id]; n > 1 {
			id = fmt.Sprintf("%s_%d", id, n)
		}
		ids[i] = id
	}
	return ids
}

// keySuffix marks a declared key: Mermaid renders PK, FK and UK as badges.
func keySuffix(col *entity.Column) string {
	switch {
	case col.PrimaryKey:
		return " PK"
	case col.References != nil:
		return " FK"
	case col.Unique:
		return " UK"
	}
	return ""
}

// cardinality is the child side of the relationship: a unique foreign key makes it one-to-(zero or one), a nullable one makes the child optional.
func cardinality(col *entity.Column) string {
	switch {
	case col.Unique:
		return "o|"
	case col.Nullable:
		return "o{"
	}
	return "|{"
}

// indexOf finds the referenced table in the listed set.
func indexOf(tables []entity.Table, ref *entity.Reference) int {
	for i := range tables {
		if tables[i].Name == ref.Table && (ref.Schema == "" || tables[i].Schema == ref.Schema) {
			return i
		}
	}
	return -1
}

// ident normalises a catalog name into a Mermaid identifier.
func ident(raw, fallback string) string {
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
