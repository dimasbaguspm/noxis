package interfaces

import (
	"context"

	"noxis/internal/entity"
	sqlsvc "noxis/internal/services/sql"
)

// SQL operation shapes live with the module; these aliases let the seam
// describe itself without duplicating types.
type (
	Result = sqlsvc.Result
	Table  = sqlsvc.Table
	Column = sqlsvc.Column
)

// SQLModule is the SQL operations seam: the console's only route to a
// database. Implemented by internal/services/sql — swapping it (tests, a
// different driver strategy) needs no handler change.
type SQLModule interface {
	Ping(ctx context.Context, c entity.Connection) error
	Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	Alter(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	DryRun(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	Apply(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (Result, error)
	Analyze(ctx context.Context, c entity.Connection, table string) (Result, error)
	Tables(ctx context.Context, c entity.Connection) ([]Table, error)
	Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (Result, error)
	VerifyWrite(sqlText string) error
}

// Satisfier: the SQL module implements its seam.
var _ SQLModule = (*sqlsvc.DB)(nil)
