package schema

import (
	"testing"

	"noxis/internal/entity"
)

// A unique name stays bare; a name repeated across schemas is qualified; unsafe characters are normalised; duplicates never collapse.
func TestAssignTableIDs(t *testing.T) {
	tables := []entity.Table{
		{Schema: "public", Name: "order.detail"},
		{Schema: "a", Name: "notes"},
		{Schema: "b", Name: "notes"},
		{Schema: "public", Name: "order detail"},
		{Schema: "public", Name: "123"},
		{Schema: "public", Name: "order_detail"},
	}
	assignTableIDs(tables)
	want := []string{"order_detail", "a_notes", "b_notes", "order_detail_2", "t_123", "order_detail_3"}
	for i, id := range want {
		if tables[i].ID != id {
			t.Fatalf("table %d id = %q, want %q", i, tables[i].ID, id)
		}
	}
}

func TestAssignTableIDsFallsBackToTable(t *testing.T) {
	tables := []entity.Table{{Schema: "public", Name: "!!!"}}
	assignTableIDs(tables)
	if tables[0].ID != "table" {
		t.Fatalf("id = %q", tables[0].ID)
	}
}
