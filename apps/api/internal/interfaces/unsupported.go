package interfaces

import (
	"context"
	"fmt"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

// Unsupported is embedded by every engine module: it answers the whole Module surface with ErrUnsupported and the verb's
// name, so a module declares only the verbs it serves and inherits an honest refusal for the rest.
//
// It deliberately does not answer Engine (Driver, Supports, Capabilities, Ping): a module's identity is its own, and a
// driver that does not say which driver it is has no business being registered.
type Unsupported struct{}

// refuse names the verb the module does not serve.
func refuse(verb string) error {
	return fmt.Errorf("%w: %s", constants.ErrUnsupported, verb)
}

// Query refuses the SQL read path.
func (Unsupported) Query(context.Context, entity.Connection, string) (Result, error) {
	return Result{}, refuse("query")
}

// DryRun refuses the SQL dry run.
func (Unsupported) DryRun(context.Context, entity.Connection, string) (Result, error) {
	return Result{}, refuse("dry run")
}

// Analyze refuses planner statistics.
func (Unsupported) Analyze(context.Context, entity.Connection, string) (Result, error) {
	return Result{}, refuse("analyze")
}

// Tables refuses schema browsing.
func (Unsupported) Tables(context.Context, entity.Connection) ([]Table, error) {
	return nil, refuse("tables")
}

// Rows refuses the row preview.
func (Unsupported) Rows(context.Context, entity.Connection, string, string, int) (Result, error) {
	return Result{}, refuse("rows")
}

// Referenced reports no tables: the change record lists what a batch touches, and a driver with no SQL touches nothing.
func (Unsupported) Referenced(string) (tables, columns []string) { return nil, nil }

// ServerInfo refuses the server report.
func (Unsupported) ServerInfo(context.Context, entity.Connection) (entity.ServerInfo, error) {
	return entity.ServerInfo{}, refuse("server info")
}

// Kind reports no kind: a driver with no SQL classifies nothing.
func (Unsupported) Kind(string) string { return "" }

// VerifyWrite refuses the write guard.
func (Unsupported) VerifyWrite(string) error { return refuse("write guard") }

// ValidateSQLChange refuses the SQL change proof.
func (Unsupported) ValidateSQLChange(context.Context, entity.Connection, string, string) (Result, error) {
	return Result{}, refuse("sql change validation")
}

// ApplySQL refuses the SQL write path.
func (Unsupported) ApplySQL(context.Context, entity.Connection, string) (Result, error) {
	return Result{}, refuse("sql apply")
}

// Keys refuses the key browse.
func (Unsupported) Keys(context.Context, entity.Connection, string, uint64, int) (KeyPage, error) {
	return KeyPage{}, refuse("keys")
}

// Value refuses reading a key's value.
func (Unsupported) Value(context.Context, entity.Connection, string) (Result, error) {
	return Result{}, refuse("value")
}

// Delete refuses removing keys.
func (Unsupported) Delete(context.Context, entity.Connection, []string) (int64, error) {
	return 0, refuse("delete")
}

// Flush refuses emptying a cache database.
func (Unsupported) Flush(context.Context, entity.Connection) error { return refuse("flush") }

// Expire refuses setting a key's expiry.
func (Unsupported) Expire(context.Context, entity.Connection, []string, int64) (int64, error) {
	return 0, refuse("expire")
}

// Info refuses the cache report.
func (Unsupported) Info(context.Context, entity.Connection) (CacheInfo, error) {
	return CacheInfo{}, refuse("cache info")
}

// Databases refuses listing document databases.
func (Unsupported) Databases(context.Context, entity.Connection) ([]MongoDatabase, error) {
	return nil, refuse("databases")
}

// Collections refuses listing collections.
func (Unsupported) Collections(context.Context, entity.Connection, string) ([]MongoCollection, error) {
	return nil, refuse("collections")
}

// Run refuses a document read.
func (Unsupported) Run(context.Context, entity.Connection, MongoRequest) (MongoResult, error) {
	return MongoResult{}, refuse("requests")
}

// ValidateRequest refuses the document change proof.
func (Unsupported) ValidateRequest(context.Context, entity.Connection, MongoRequest) (MongoResult, error) {
	return MongoResult{}, refuse("document change validation")
}

// ApplyDocument refuses the document write path.
func (Unsupported) ApplyDocument(context.Context, entity.Connection, MongoRequest) (MongoResult, error) {
	return MongoResult{}, refuse("document apply")
}

// IsWrite reports false: a driver with no document requests mutates nothing.
func (Unsupported) IsWrite(string) bool { return false }

// Targets refuses listing search targets.
func (Unsupported) Targets(context.Context, entity.Connection) ([]SearchTarget, error) {
	return nil, refuse("targets")
}

// Search refuses a search query.
func (Unsupported) Search(context.Context, entity.Connection, SearchRequest) (Result, error) {
	return Result{}, refuse("search")
}

// ValidateSearchChange refuses the search change proof.
func (Unsupported) ValidateSearchChange(context.Context, entity.Connection, SearchRequest) (Result, error) {
	return Result{}, refuse("search change validation")
}

// ApplySearch refuses the search write path.
func (Unsupported) ApplySearch(context.Context, entity.Connection, SearchRequest) (Result, error) {
	return Result{}, refuse("search apply")
}
