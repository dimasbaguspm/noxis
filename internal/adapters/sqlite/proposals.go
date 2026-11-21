package sqlite

import (
	"context"
	"fmt"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
	"noxis/internal/ports"
	"strings"
	"time"
)

const proposalColumns = `ref, target_ref, title, kind, query, rollback, note, status, tier, irreversible,
	proposed_by, rehearsal_ref, decided_by, decided_at, decided_note, applied_by, applied_at, failure,
	started_at, progress_stage, progress_percent, algorithm, rows_estimated, rows_affected,
	created_at, updated_at, idempotency_key`

// Create stores one proposal. Two proposals with the same title are allowed: a title describes a
// change, and the same change may be proposed twice.
func (d *DB) CreateProposal(ctx context.Context, proposal domain.Proposal) (out0 domain.Proposal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.CreateProposal")
	defer func() { end(err) }()

	_, err = d.db.ExecContext(ctx, `INSERT INTO proposals (`+proposalColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		proposal.Ref, proposal.TargetRef, proposal.Title, string(proposal.Kind), proposal.Query,
		proposal.Rollback, proposal.Note, string(proposal.Status), string(proposal.Tier),
		boolInt(proposal.Irreversible), proposal.ProposedBy, proposal.RehearsalRef, proposal.DecidedBy,
		nullableMillis(proposal.DecidedAt), proposal.DecidedNote, proposal.AppliedBy,
		nullableMillis(proposal.AppliedAt), proposal.Failure, nullableMillis(proposal.StartedAt),
		proposal.Progress.Stage, proposal.Progress.Percent, string(proposal.Algorithm),
		proposal.RowsEstimated, proposal.RowsAffected, millis(proposal.CreatedAt),
		millis(proposal.UpdatedAt), nullableKey(proposal.IdempotencyKey),
	)
	if err != nil {
		if isUniqueViolation(err) && proposal.IdempotencyKey != "" {
			// Two creations racing on one key: the loser answers the winner's row.
			if existing, lookupErr := d.ProposalByKey(ctx, proposal.IdempotencyKey); lookupErr == nil {
				return existing, nil
			}
		}
		return domain.Proposal{}, fmt.Errorf("state: create proposal: %w", err)
	}
	return proposal, nil
}

// ProposalByKey answers the proposal a retried create already made, or ErrNotFound.
func (d *DB) ProposalByKey(ctx context.Context, key string) (out0 domain.Proposal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.ProposalByKey")
	defer func() { end(err) }()

	row := d.db.QueryRowContext(ctx, `SELECT `+proposalColumns+` FROM proposals WHERE idempotency_key = ?`, key)
	proposal, err := scanProposal(row)
	if err != nil {
		return domain.Proposal{}, notFound(err, "the idempotency key", key)
	}
	return proposal, nil
}

// ByRef reads one proposal.
func (d *DB) ProposalByRef(ctx context.Context, ref string) (out0 domain.Proposal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.ProposalByRef")
	defer func() { end(err) }()

	row := d.db.QueryRowContext(ctx, `SELECT `+proposalColumns+` FROM proposals WHERE ref = ?`, ref)
	proposal, err := scanProposal(row)
	if err != nil {
		return domain.Proposal{}, notFound(err, "proposal", ref)
	}
	return proposal, nil
}

// List answers one page of the queue, newest first.
func (d *DB) ListProposals(ctx context.Context, filter ports.ProposalFilter) (out0 []domain.Proposal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.ListProposals")
	defer func() { end(err) }()

	var (
		where []string
		args  []any
	)
	if filter.Target != "" {
		where = append(where, "target_ref = ?")
		args = append(args, filter.Target)
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, filter.Kind)
	}
	if filter.Mine != "" {
		where = append(where, "proposed_by = ?")
		args = append(args, filter.Mine)
	}
	if filter.Cursor != "" {
		where = append(where, "ref < ?")
		args = append(args, filter.Cursor)
	}

	query := `SELECT ` + proposalColumns + ` FROM proposals`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY ref DESC LIMIT ?"
	args = append(args, filter.Limit)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("state: list proposals: %w", err)
	}
	defer func() { _ = rows.Close() }()

	proposals := make([]domain.Proposal, 0, filter.Limit)
	for rows.Next() {
		proposal, err := scanProposal(rows)
		if err != nil {
			return nil, err
		}
		proposals = append(proposals, proposal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: list proposals: %w", err)
	}
	return proposals, nil
}

// Update writes a proposal back. A missing row is reported rather than silently ignored.
func (d *DB) UpdateProposal(ctx context.Context, proposal domain.Proposal) (out0 domain.Proposal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.UpdateProposal")
	defer func() { end(err) }()

	res, err := d.db.ExecContext(ctx, `UPDATE proposals SET
			title = ?, kind = ?, query = ?, rollback = ?, note = ?, status = ?, tier = ?,
			irreversible = ?, rehearsal_ref = ?, decided_by = ?, decided_at = ?, decided_note = ?,
			applied_by = ?, applied_at = ?, failure = ?, started_at = ?, progress_stage = ?,
			progress_percent = ?, algorithm = ?, rows_estimated = ?, rows_affected = ?, updated_at = ?
		WHERE ref = ?`,
		proposal.Title, string(proposal.Kind), proposal.Query, proposal.Rollback, proposal.Note,
		string(proposal.Status), string(proposal.Tier), boolInt(proposal.Irreversible),
		proposal.RehearsalRef, proposal.DecidedBy, nullableMillis(proposal.DecidedAt),
		proposal.DecidedNote, proposal.AppliedBy, nullableMillis(proposal.AppliedAt),
		proposal.Failure, nullableMillis(proposal.StartedAt), proposal.Progress.Stage,
		proposal.Progress.Percent, string(proposal.Algorithm), proposal.RowsEstimated,
		proposal.RowsAffected, millis(proposal.UpdatedAt), proposal.Ref,
	)
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("state: update proposal: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("state: update proposal: %w", err)
	}
	if affected == 0 {
		return domain.Proposal{}, fmt.Errorf("%w: proposal %s", domain.ErrNotFound, proposal.Ref)
	}
	return proposal, nil
}

func scanProposal(row scanner) (domain.Proposal, error) {
	var (
		proposal       domain.Proposal
		kind           string
		status         string
		tier           string
		irrev          int
		decidedAt      *int64
		appliedAt      *int64
		startedAt      *int64
		algorithm      string
		createdAt      int64
		updatedAt      int64
		idempotencyKey *string
	)
	err := row.Scan(&proposal.Ref, &proposal.TargetRef, &proposal.Title, &kind, &proposal.Query,
		&proposal.Rollback, &proposal.Note, &status, &tier, &irrev, &proposal.ProposedBy,
		&proposal.RehearsalRef, &proposal.DecidedBy, &decidedAt, &proposal.DecidedNote,
		&proposal.AppliedBy, &appliedAt, &proposal.Failure, &startedAt, &proposal.Progress.Stage,
		&proposal.Progress.Percent, &algorithm, &proposal.RowsEstimated, &proposal.RowsAffected,
		&createdAt, &updatedAt, &idempotencyKey)
	if err != nil {
		return domain.Proposal{}, err
	}
	proposal.Kind = domain.Kind(kind)
	proposal.Status = domain.Status(status)
	proposal.Tier = domain.Tier(tier)
	proposal.Irreversible = irrev == 1
	if decidedAt != nil {
		at := fromMillis(*decidedAt)
		proposal.DecidedAt = &at
	}
	if appliedAt != nil {
		at := fromMillis(*appliedAt)
		proposal.AppliedAt = &at
	}
	if startedAt != nil {
		at := fromMillis(*startedAt)
		proposal.StartedAt = &at
	}
	proposal.Algorithm = domain.Algorithm(algorithm)
	proposal.CreatedAt = fromMillis(createdAt)
	proposal.UpdatedAt = fromMillis(updatedAt)
	if idempotencyKey != nil {
		proposal.IdempotencyKey = *idempotencyKey
	}
	return proposal, nil
}

// Create stores one rehearsal attempt.
func (d *DB) CreateRehearsal(ctx context.Context, rehearsal domain.Rehearsal) (out0 domain.Rehearsal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.CreateRehearsal")
	defer func() { end(err) }()

	return d.insertRehearsal(ctx, rehearsal)
}

// ByRef reads one rehearsal.
func (d *DB) RehearsalByRef(ctx context.Context, ref string) (out0 domain.Rehearsal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.RehearsalByRef")
	defer func() { end(err) }()

	row := d.db.QueryRowContext(ctx, `SELECT `+rehearsalColumns+` FROM rehearsals WHERE ref = ?`, ref)
	rehearsal, err := scanRehearsal(row)
	if err != nil {
		return domain.Rehearsal{}, notFound(err, "rehearsal", ref)
	}
	return rehearsal, nil
}

// Latest reads the most recent rehearsal of one proposal.
func (d *DB) LatestRehearsal(ctx context.Context, proposalRef string) (out0 domain.Rehearsal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.LatestRehearsal")
	defer func() { end(err) }()

	row := d.db.QueryRowContext(ctx, `SELECT `+rehearsalColumns+`
		FROM rehearsals WHERE proposal_ref = ? ORDER BY ref DESC LIMIT 1`, proposalRef)
	rehearsal, err := scanRehearsal(row)
	if err != nil {
		return domain.Rehearsal{}, notFound(err, "rehearsal for proposal", proposalRef)
	}
	return rehearsal, nil
}

const rehearsalColumns = `ref, proposal_ref, tier, status, schema_diff, row_diff, plan, irreversible,
	note, duration_ms, error, created_at`

func (d *DB) insertRehearsal(ctx context.Context, rehearsal domain.Rehearsal) (out0 domain.Rehearsal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.insertRehearsal")
	defer func() { end(err) }()

	_, err = d.db.ExecContext(ctx, `INSERT INTO rehearsals (`+rehearsalColumns+`)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		rehearsal.Ref, rehearsal.ProposalRef, string(rehearsal.Tier), string(rehearsal.Status),
		rehearsal.SchemaDiff, rehearsal.RowDiff, rehearsal.Plan, boolInt(rehearsal.Irreversible),
		rehearsal.Note, rehearsal.DurationMS, rehearsal.Error, millis(rehearsal.CreatedAt),
	)
	if err != nil {
		return domain.Rehearsal{}, fmt.Errorf("state: create rehearsal: %w", err)
	}
	return rehearsal, nil
}

func scanRehearsal(row scanner) (domain.Rehearsal, error) {
	var (
		rehearsal domain.Rehearsal
		tier      string
		status    string
		irrev     int
		createdAt int64
	)
	err := row.Scan(&rehearsal.Ref, &rehearsal.ProposalRef, &tier, &status, &rehearsal.SchemaDiff,
		&rehearsal.RowDiff, &rehearsal.Plan, &irrev, &rehearsal.Note, &rehearsal.DurationMS,
		&rehearsal.Error, &createdAt)
	if err != nil {
		return domain.Rehearsal{}, err
	}
	rehearsal.Tier = domain.Tier(tier)
	rehearsal.Status = domain.RehearsalStatus(status)
	rehearsal.Irreversible = irrev == 1
	rehearsal.CreatedAt = fromMillis(createdAt)
	return rehearsal, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// compile-time proof the store satisfies both seams.
var (
	_ ports.Proposals  = (*DB)(nil)
	_ ports.Rehearsals = (*DB)(nil)
)

// SetProgress records what the engine said about the running statement. It touches the progress
// columns and nothing else: the receipt that ends the apply is written by the use case, and the two
// must not overwrite each other.
func (d *DB) SetProgress(ctx context.Context, ref string, progress domain.Progress) error {
	_, err := d.db.ExecContext(ctx, `UPDATE proposals SET progress_stage = ?, progress_percent = ?
		WHERE ref = ?`, progress.Stage, progress.Percent, ref)
	if err != nil {
		return fmt.Errorf("state: set progress: %w", err)
	}
	return nil
}

// ClaimProposal moves a proposal from one status to another only while it is still there. The
// conditional update is the lock: whoever's update reports a row won the claim, everyone else learns
// the proposal is no longer theirs to run.
func (d *DB) ClaimProposal(ctx context.Context, ref string, from, to domain.Status) (out0 domain.Proposal, err error) {
	ctx, end := telemetry.Span(ctx, "sqlite.DB.ClaimProposal")
	defer func() { end(err) }()

	res, err := d.db.ExecContext(ctx, `UPDATE proposals SET status = ?, updated_at = ?
		WHERE ref = ? AND status = ?`, string(to), millis(time.Now().UTC()), ref, string(from))
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("state: claim proposal: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("state: claim proposal: %w", err)
	}
	if affected == 0 {
		current, err := d.ProposalByRef(ctx, ref)
		if err != nil {
			return domain.Proposal{}, err
		}
		return current, fmt.Errorf("%w: proposal is %s, not %s", domain.ErrConflict, current.Status, from)
	}
	return d.ProposalByRef(ctx, ref)
}
