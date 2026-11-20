package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"noxis/internal/common/sqltext"
	"noxis/internal/domain"
)

// Plan sizes a change before it runs. The number is the engine's own estimate from
// information_schema — the same one the optimizer reads — because counting the table to answer
// "how long will this take" would cost as much as the change.
func (d *DB) Plan(ctx context.Context, target domain.Target, password string, change domain.Change) (domain.Plan, error) {
	plan := domain.Plan{}
	statement, err := sqltext.Classify(change.Query)
	if err != nil || len(statement.Tables) == 0 {
		// Nothing to size: a statement that names no table is not the one this decision is about.
		return plan, nil
	}
	named := sqltext.ParseQualified(statement.Tables[0])
	database := named.Database
	if database == "" {
		database = target.Database
	}

	pool, err := d.pool(target, password)
	if err != nil {
		return plan, err
	}
	defer func() { _ = pool.Close() }()

	err = pool.QueryRowContext(ctx,
		`SELECT COALESCE(TABLE_ROWS, 0) FROM information_schema.TABLES
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, database, named.Table).Scan(&plan.RowsEstimated)
	if errors.Is(err, sql.ErrNoRows) {
		return plan, nil
	}
	if err != nil {
		return plan, statementError(err, password)
	}
	plan.Algorithm = expectedAlgorithm(statement)
	return plan, nil
}

// expectedAlgorithm is what the engine will probably choose, and it is a guess: MySQL 8 and MariaDB
// pick at run time, and the receipt records what actually happened. The guess is only used to tell a
// reviewer what kind of work to expect, never to promise it.
func expectedAlgorithm(statement sqltext.Statement) domain.Algorithm {
	raw := strings.ToUpper(statement.Raw)
	switch {
	case strings.Contains(raw, "ADD COLUMN"), strings.Contains(raw, "DROP COLUMN"):
		return domain.AlgorithmInstant
	case strings.Contains(raw, "CREATE INDEX"), strings.Contains(raw, "ADD INDEX"),
		strings.Contains(raw, "ADD KEY"), strings.Contains(raw, "DROP INDEX"):
		return domain.AlgorithmInPlace
	}
	return ""
}
