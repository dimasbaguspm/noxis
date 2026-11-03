package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// Engine is what every engine module is, whatever it can do: the driver it serves, the ping that proves a connection, and
// the capabilities it declares.
type Engine interface {
	// Driver is the driver this module serves; the API namespace is the driver name.
	Driver() string
	// Supports reports whether this module serves a driver, which keeps a connection out of another driver's namespace.
	Supports(driver string) bool
	// Capabilities lists the surface this module actually serves, from the constants.Capability* values. The console
	// groups its navigation by them, so a driver the frontend has never seen still appears under the group it serves.
	Capabilities() []string
	Ping(ctx context.Context, c entity.Connection) error
}

// Module is the whole engine contract: every verb the API can ask of a driver, whatever family it belongs to.
//
// One module answers all of them. The verbs a driver does not serve are inherited from Unsupported, which refuses with
// ErrUnsupported and the verb's name, so a caller reads why instead of meeting a missing method. A family is declared
// once, in Capabilities, and proved by the narrow seams below.
type Module interface {
	Engine

	// SQL.

	Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	DryRun(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	Analyze(ctx context.Context, c entity.Connection, table string) (Result, error)
	Tables(ctx context.Context, c entity.Connection) ([]Table, error)
	Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (Result, error)
	Referenced(sqlText string) (tables []string, columns []string)
	ServerInfo(ctx context.Context, c entity.Connection) (entity.ServerInfo, error)
	Kind(sqlText string) string
	VerifyWrite(sqlText string) error
	// ValidateSQLChange dry-runs a query and its rollback together.
	ValidateSQLChange(ctx context.Context, c entity.Connection, query, rollback string) (Result, error)
	// ApplySQL runs an approved SQL batch for real.
	ApplySQL(ctx context.Context, c entity.Connection, sqlText string) (Result, error)

	// Memory cache.

	Keys(ctx context.Context, c entity.Connection, pattern string, cursor uint64, limit int) (KeyPage, error)
	Value(ctx context.Context, c entity.Connection, key string) (Result, error)
	Delete(ctx context.Context, c entity.Connection, keys []string) (int64, error)
	Flush(ctx context.Context, c entity.Connection) error
	Expire(ctx context.Context, c entity.Connection, keys []string, ttlMs int64) (int64, error)
	Info(ctx context.Context, c entity.Connection) (CacheInfo, error)

	// Documents.

	Databases(ctx context.Context, c entity.Connection) ([]MongoDatabase, error)
	Collections(ctx context.Context, c entity.Connection, database string) ([]MongoCollection, error)
	Run(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	ValidateRequest(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	// ApplyDocument runs an approved document write for real.
	ApplyDocument(ctx context.Context, c entity.Connection, req MongoRequest) (MongoResult, error)
	IsWrite(op string) bool

	// Search platforms.

	Targets(ctx context.Context, c entity.Connection) ([]SearchTarget, error)
	Search(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// ValidateSearchChange proves a search write without applying it.
	ValidateSearchChange(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
	// ApplySearch runs an approved search write for real.
	ApplySearch(ctx context.Context, c entity.Connection, req SearchRequest) (Result, error)
}
