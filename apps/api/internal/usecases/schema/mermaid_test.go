package schema

import (
	"strings"
	"testing"

	"noxis/internal/entity"
)

// people and pets: one declared primary key each and one declared foreign key.
func sample() []entity.Table {
	return []entity.Table{
		{Schema: "public", Name: "people", Columns: []entity.Column{
			{Name: "id", Type: "integer", PrimaryKey: true},
			{Name: "name", Type: "character varying(64)"},
		}},
		{Schema: "public", Name: "pets", Columns: []entity.Column{
			{Name: "id", Type: "integer", PrimaryKey: true},
			{Name: "owner_id", Type: "integer", References: &entity.Reference{Schema: "public", Table: "people", Column: "id"}},
		}},
	}
}

func TestMermaidRendersTablesKeysAndRelationships(t *testing.T) {
	got := Mermaid(sample())
	for _, want := range []string{
		"erDiagram\n",
		"    people {\n",
		"        integer id PK\n",
		"        character_varying_64 name\n",
		"    pets {\n",
		"        integer owner_id FK\n",
		// a non-null foreign key: exactly one parent, one or more children
		"    people ||--|{ pets : \"owner_id\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mermaid missing %q:\n%s", want, got)
		}
	}
}

// Only declared constraints are drawn: a column named owner_id with no foreign key in the catalog is just a column.
func TestMermaidIgnoresUndeclaredReferences(t *testing.T) {
	tables := []entity.Table{{Schema: "public", Name: "pets", Columns: []entity.Column{
		{Name: "owner_id", Type: "integer"},
	}}}
	got := Mermaid(tables)
	if strings.Contains(got, "||--") {
		t.Fatalf("no relationship must be guessed from a column name:\n%s", got)
	}
}

// A schema-qualified name is normalised, a name repeated across schemas keeps its schema, and a type with a length does not break the identifier rules.
func TestMermaidNormalisesNames(t *testing.T) {
	tables := []entity.Table{
		{Schema: "public", Name: "order.detail", Columns: []entity.Column{{Name: "line 1", Type: "numeric(10,2)"}}},
		{Schema: "a", Name: "notes", Columns: []entity.Column{{Name: "id", Type: "int"}}},
		{Schema: "b", Name: "notes", Columns: []entity.Column{{Name: "id", Type: "int"}}},
	}
	got := Mermaid(tables)
	for _, want := range []string{
		"    order_detail {\n", "        numeric_10_2 line_1\n",
		"    a_notes {\n", "    b_notes {\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mermaid missing %q:\n%s", want, got)
		}
	}
}

// A foreign key to a table the connection does not list is skipped rather than drawn against a non-existent entity.
func TestMermaidSkipsUnknownParent(t *testing.T) {
	tables := []entity.Table{{Schema: "public", Name: "pets", Columns: []entity.Column{
		{Name: "owner_id", Type: "integer", References: &entity.Reference{Schema: "auth", Table: "users", Column: "id"}},
	}}}
	if got := Mermaid(tables); strings.Contains(got, "||--") {
		t.Fatalf("a reference outside the listed schema must not be drawn:\n%s", got)
	}
}

// A unique foreign key is one-to-one, a nullable one is optional.
func TestMermaidCardinality(t *testing.T) {
	cases := map[string]struct {
		col  entity.Column
		want string
	}{
		"unique":    {entity.Column{Name: "k", Unique: true}, "||--o|"},
		"nullable":  {entity.Column{Name: "k", Nullable: true}, "||--o{"},
		"mandatory": {entity.Column{Name: "k"}, "||--|{"},
	}
	for name, tc := range cases {
		tc.col.References = &entity.Reference{Schema: "public", Table: "people", Column: "id"}
		tables := []entity.Table{
			{Schema: "public", Name: "people", Columns: []entity.Column{{Name: "id", Type: "int", PrimaryKey: true}}},
			{Schema: "public", Name: "child", Columns: []entity.Column{tc.col}},
		}
		got := Mermaid(tables)
		if !strings.Contains(got, tc.want+" child") {
			t.Fatalf("%s: want %q in:\n%s", name, tc.want, got)
		}
	}
}

func TestMermaidEmpty(t *testing.T) {
	if got := Mermaid(nil); got != "erDiagram\n" {
		t.Fatalf("empty graph = %q", got)
	}
}
