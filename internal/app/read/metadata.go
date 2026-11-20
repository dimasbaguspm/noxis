package read

import (
	"context"
	"fmt"
	"strings"

	"noxis/internal/domain"
)

// Databases lists the schemas one target can see.
type Databases struct{ Deps }

// NewDatabases builds the use case.
func NewDatabases(d Deps) *Databases { return &Databases{Deps: d} }

// Run reads the schema list.
func (d *Databases) Run(ctx context.Context, ref string) ([]string, error) {
	target, password, err := d.connect(ctx, ref)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()
	return d.Driver.Databases(ctx, target, password)
}

// Tables lists the tables in one schema.
type Tables struct{ Deps }

// NewTables builds the use case.
func NewTables(d Deps) *Tables { return &Tables{Deps: d} }

// TablesInput names the schema.
type TablesInput struct {
	Ref      string
	Database string
}

// Run reads the table list.
func (t *Tables) Run(ctx context.Context, in TablesInput) ([]domain.Table, error) {
	if strings.TrimSpace(in.Database) == "" {
		return nil, fmt.Errorf("%w: a schema name is required", domain.ErrInvalid)
	}
	target, password, err := t.connect(ctx, in.Ref)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()
	return t.Driver.Tables(ctx, target, password, in.Database)
}

// Table describes one table: its columns, its indexes and the DDL the engine would use.
type Table struct{ Deps }

// NewTable builds the use case.
func NewTable(d Deps) *Table { return &Table{Deps: d} }

// TableInput names the schema and the table.
type TableInput struct {
	Ref      string
	Database string
	Table    string
}

// Run reads one table's definition.
func (t *Table) Run(ctx context.Context, in TableInput) (domain.TableDetail, error) {
	if strings.TrimSpace(in.Database) == "" || strings.TrimSpace(in.Table) == "" {
		return domain.TableDetail{}, fmt.Errorf("%w: a schema and a table name are required", domain.ErrInvalid)
	}
	target, password, err := t.connect(ctx, in.Ref)
	if err != nil {
		return domain.TableDetail{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()
	return t.Driver.Table(ctx, target, password, in.Database, in.Table)
}
