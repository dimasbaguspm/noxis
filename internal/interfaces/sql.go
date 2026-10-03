package interfaces

import (
	"context"

	"noxis/internal/entity"
	sqlsvc "noxis/internal/services/sql"
)

// SQL operation shapes live with the module; the result grid is shared.
type (
	Result = entity.Result
	Table  = sqlsvc.Table
	Column = sqlsvc.Column
	Stats  = sqlsvc.Stats
)

// Module is what any engine family must provide for a connection to be
// registered and tested. Adding an engine starts here.
type Module interface {
	// Family is the API namespace this module serves (constants.FamilySQL,
	// constants.FamilyRedis).
	Family() string
	// Supports reports whether this module serves the driver, which keeps a
	// connection out of the other families' namespaces.
	Supports(driver string) bool
	Ping(ctx context.Context, c entity.Connection) error
}

// SQLModule is the console's only route to a SQL database.
type SQLModule interface {
	Module
	Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	Alter(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	DryRun(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	Apply(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (Result, error)
	Analyze(ctx context.Context, c entity.Connection, table string) (Result, error)
	Tables(ctx context.Context, c entity.Connection) ([]Table, error)
	Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (Result, error)
	VerifyWrite(sqlText string) error
	Kind(sqlText string) string
	Stats(ctx context.Context, c entity.Connection) (Stats, error)
}

// Satisfier.
var _ SQLModule = (*sqlsvc.DB)(nil)
