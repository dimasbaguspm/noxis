package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"noxis/internal/common/sqltext"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strings"
	"time"
)

// Apply runs an approved statement for real. A DML statement is wrapped in one transaction; a DDL
// statement cannot be, because both engines commit it implicitly, and pretending otherwise would
// print a receipt for a rollback that never happened.
//
// The statement runs on one connection of its own so the watcher can ask the server about the
// thread that runs it, and the receipt carries what the engine actually did: which algorithm it
// chose, and how many rows it touched.
func (d *DB) Apply(ctx context.Context, target domain.Target, password string, change domain.Change,
	report func(domain.Progress),
) (out domain.Receipt, err error) {
	ctx, end := telemetry.Span(ctx, "mysql.DB.Apply")
	defer func() { end(err) }()
	began := time.Now()
	defer func() { telemetry.Apply(ctx, telemetry.Outcome(err), target.Driver, time.Since(began).Seconds()) }()

	statement, err := sqltext.Classify(change.Query)
	if err != nil {
		return out, statementError(err, password)
	}

	pool, err := d.pool(target, password)
	if err != nil {
		return out, err
	}
	defer func() { _ = pool.Close() }()

	conn, err := pool.Conn(ctx)
	if err != nil {
		return out, statementError(err, password)
	}
	defer func() { _ = conn.Close() }()

	stop := d.watch(ctx, pool, conn, change, report, &out)
	defer stop()

	if statement.Kind == sqltext.DDL {
		// How many instant changes the table has taken, before this one. A statement that finishes
		// between two samples leaves no stage to read, and this counter is what still answers.
		versionsBefore := d.rowVersions(ctx, conn, target, change)
		res, execErr := conn.ExecContext(ctx, change.Query)
		if execErr != nil {
			return out, statementError(execErr, password)
		}
		out.RowsAffected, _ = res.RowsAffected()
		if out.Algorithm == "" && versionsBefore >= 0 {
			if versionsAfter := d.rowVersions(ctx, conn, target, change); versionsAfter > versionsBefore {
				out.Algorithm = domain.AlgorithmInstant
			}
		}
		return out, nil
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return out, statementError(err, password)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, change.Query)
	if err != nil {
		return out, statementError(err, password)
	}
	out.RowsAffected, _ = res.RowsAffected()
	return out, statementError(tx.Commit(), password)
}

// rowVersions reads how many instant column changes a table has taken. It returns -1 when the
// engine will not say, so a caller can tell "nothing changed" from "nobody answered".
func (*DB) rowVersions(ctx context.Context, conn *sql.Conn, target domain.Target, change domain.Change) int64 {
	statement, err := sqltext.Classify(change.Query)
	if err != nil || len(statement.Tables) == 0 {
		return -1
	}
	named := sqltext.ParseQualified(statement.Tables[0])
	database := named.Database
	if database == "" {
		database = target.Database
	}
	var versions int64
	err = conn.QueryRowContext(ctx,
		`SELECT COALESCE(TOTAL_ROW_VERSIONS, 0) FROM information_schema.INNODB_TABLES WHERE NAME = ?`,
		fmt.Sprintf("%s/%s", database, named.Table)).Scan(&versions)
	if err != nil {
		return -1
	}
	return versions
}

// watch asks the server what the running statement is doing, every progress interval, and stops
// when the returned function is called. It never fails the apply: progress is an observation, and an
// observation that fails is a missing percentage, not a broken change.
func (*DB) watch(ctx context.Context, pool *sql.DB, conn *sql.Conn, _ domain.Change,
	report func(domain.Progress), receipt *domain.Receipt,
) func() {
	var threadID int64
	if err := conn.QueryRowContext(ctx, "SELECT CONNECTION_ID()").Scan(&threadID); err != nil {
		return func() {}
	}

	watched, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(constants.ApplyProgressInterval)
		defer ticker.Stop()
		for {
			select {
			case <-watched.Done():
				return
			case <-ticker.C:
				stage, percent := stageOf(watched, pool, threadID)
				if stage == "" && percent < 0 {
					continue
				}
				if algorithm := algorithmOf(stage); algorithm != "" {
					receipt.Algorithm = algorithm
				}
				if report != nil {
					report(domain.Progress{Stage: stage, Percent: percent})
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

// stageOf reads the stage the statement is in. performance_schema knows the stage and, for a copy,
// how much of it is done; the process list knows less but is never disabled, which is what a server
// with performance_schema switched off still answers.
func stageOf(ctx context.Context, pool *sql.DB, threadID int64) (string, int) {
	var (
		stage     string
		completed int64
		estimated int64
	)
	err := pool.QueryRowContext(ctx,
		`SELECT COALESCE(EVENT_NAME, ''), COALESCE(WORK_COMPLETED, 0), COALESCE(WORK_ESTIMATED, 0)
			FROM performance_schema.events_stages_current WHERE THREAD_ID = ?`,
		threadID).Scan(&stage, &completed, &estimated)
	if err == nil && stage != "" {
		percent := -1
		if estimated > 0 {
			percent = int(completed * 100 / estimated)
			if percent > 100 {
				percent = 100
			}
		}
		return stage, percent
	}

	var state string
	if err := pool.QueryRowContext(ctx,
		`SELECT COALESCE(STATE, '') FROM information_schema.PROCESSLIST WHERE ID = ?`,
		threadID).Scan(&state); err != nil {
		return "", -1
	}
	return state, -1
}

// algorithmOf names the algorithm from the stage the server reported. An empty answer means the
// server did not say, which is stored as unknown rather than guessed.
func algorithmOf(stage string) domain.Algorithm {
	lowered := strings.ToLower(stage)
	switch {
	case strings.Contains(lowered, "copy to tmp table"), strings.Contains(lowered, "copying to tmp table"):
		return domain.AlgorithmCopy
	case strings.Contains(lowered, "instant"):
		return domain.AlgorithmInstant
	case strings.Contains(lowered, "in place"), strings.Contains(lowered, "rebuild"):
		return domain.AlgorithmInPlace
	}
	return ""
}
