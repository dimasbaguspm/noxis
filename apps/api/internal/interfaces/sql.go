package interfaces

import (
	"context"
	"errors"

	"noxis/internal/entity"
)

// SQL shapes are shared wire types; the result grid is shared with MongoDB.
type (
	Result = entity.Result
	Table  = entity.Table
	Column = entity.Column
)

// Module is what every engine module provides for a connection to be registered and tested: its driver name and a ping.
type Module interface {
	// Driver is the driver this module serves; the API namespace is the driver name.
	Driver() string
	// Supports reports whether this module serves a driver, which keeps a connection out of another driver's namespace.
	Supports(driver string) bool
	Ping(ctx context.Context, c entity.Connection) error
}

// ConsoleQuerier runs the direct read operations a console page offers.
type ConsoleQuerier interface {
	Module
	Query(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	DryRun(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
	Analyze(ctx context.Context, c entity.Connection, table string) (Result, error)
}

// SchemaReader lists the tables a connection declares and the rows of one table.
type SchemaReader interface {
	Module
	Tables(ctx context.Context, c entity.Connection) ([]Table, error)
	Rows(ctx context.Context, c entity.Connection, schema, table string, limit int) (Result, error)
}

// ServerReporter reports what the connected server says about itself: engine, detected version, and what that profile can prove.
type ServerReporter interface {
	Module
	ServerInfo(ctx context.Context, c entity.Connection) (entity.ServerInfo, error)
}

// ChangeProver decides whether a batch may be proposed: which statements it carries, what it touches, and whether its rollback undoes it.
type ChangeProver interface {
	Module
	Kind(sqlText string) string
	Referenced(sqlText string) (tables []string, columns []string)
	VerifyWrite(sqlText string) error
	ValidateChange(ctx context.Context, c entity.Connection, query, rollback string) (Result, error)
}

// ChangeApplier runs an approved batch for real.
type ChangeApplier interface {
	Module
	Apply(ctx context.Context, c entity.Connection, sqlText string) (Result, error)
}

// SQLChangeEngine is the whole SQL engine a change queue drives: it proves and applies, so the queue owns no SQL itself.
type SQLChangeEngine interface {
	ChangeProver
	ChangeApplier
}

// ErrUnverifiable reports that a change cannot be dry-run on this driver (mysql commits DDL implicitly), so validation is skipped honestly.
var ErrUnverifiable = errors.New("unverifiable on this driver")

// IsUnverifiable reports the driver-cannot-dry-run case.
func IsUnverifiable(err error) bool { return errors.Is(err, ErrUnverifiable) }

// Unverifiable names why a change could not be dry-run: the reason is a code the queue can group by, and the detail is
// the one line a reviewer reads.
func Unverifiable(reason, detail string) error {
	return &unverifiable{reason: reason, detail: detail}
}

// UnverifiableReason reports the code an unverifiable error carries, or "" when it carries none.
func UnverifiableReason(err error) string {
	var target *unverifiable
	if errors.As(err, &target) {
		return target.reason
	}
	return ""
}

// unverifiable is the sentinel plus its reason, so a caller reads why without parsing prose.
type unverifiable struct {
	reason string
	detail string
}

func (u *unverifiable) Error() string { return ErrUnverifiable.Error() + ": " + u.detail }

// Is keeps errors.Is(err, ErrUnverifiable) true whatever the reason.
func (u *unverifiable) Is(target error) bool { return target == ErrUnverifiable }
