package mysql

import (
	"context"
	"time"

	"noxis/internal/common/sqltext"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
)

// Apply runs an approved statement for real. A DML statement is wrapped in one transaction; a DDL
// statement cannot be, because both engines commit it implicitly, and pretending otherwise would
// print a receipt for a rollback that never happened.
func (d *DB) Apply(ctx context.Context, target domain.Target, password string, change domain.Change) (err error) {
	ctx, end := telemetry.Span(ctx, "mysql.DB.Apply")
	defer func() { end(err) }()
	began := time.Now()
	defer func() { telemetry.Apply(ctx, telemetry.Outcome(err), target.Driver, time.Since(began).Seconds()) }()
	pool, err := d.pool(target, password)
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()

	statement, err := sqltext.Classify(change.Query)
	if err != nil {
		return statementError(err, password)
	}
	if statement.Kind == sqltext.DDL {
		if _, err := pool.ExecContext(ctx, change.Query); err != nil {
			return statementError(err, password)
		}
		return nil
	}

	tx, err := pool.BeginTx(ctx, nil)
	if err != nil {
		return statementError(err, password)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, change.Query); err != nil {
		return statementError(err, password)
	}
	return statementError(tx.Commit(), password)
}
