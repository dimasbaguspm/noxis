package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"noxis/internal/domain"
	"noxis/internal/ports"
)

const targetColumns = `ref, name, driver, host, port, database, username, password_cipher, tls_mode, options, created_by, archived_at, created_at, updated_at`

// Create inserts one target. A name collision is a conflict, not an overwrite.
func (d *DB) Create(ctx context.Context, target domain.Target) (domain.Target, error) {
	options, err := encodeOptions(target.Options)
	if err != nil {
		return domain.Target{}, err
	}
	_, err = d.db.ExecContext(ctx, `INSERT INTO targets (`+targetColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		target.Ref, target.Name, target.Driver, target.Host, target.Port, target.Database,
		target.Username, target.Secret, target.TLSMode, options, target.CreatedBy,
		nullableMillis(target.ArchivedAt), millis(target.CreatedAt), millis(target.UpdatedAt),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Target{}, fmt.Errorf("%w: a target named %q already exists", domain.ErrConflict, target.Name)
		}
		return domain.Target{}, fmt.Errorf("state: create target: %w", err)
	}
	return target, nil
}

// ByRef reads one target by its reference.
func (d *DB) ByRef(ctx context.Context, ref string) (domain.Target, error) {
	row := d.db.QueryRowContext(ctx, `SELECT `+targetColumns+` FROM targets WHERE ref = ?`, ref)
	target, err := scanTarget(row)
	if err != nil {
		return domain.Target{}, notFound(err, "target", ref)
	}
	return target, nil
}

// List answers a page. Ordering is `ref DESC` — the UUIDv7 encodes creation time, so newest
// first and the keyset are the same column.
func (d *DB) List(ctx context.Context, filter ports.TargetFilter) ([]domain.Target, error) {
	var (
		where []string
		args  []any
	)
	if !filter.IncludeArchived {
		where = append(where, "archived_at IS NULL")
	}
	if filter.Driver != "" {
		where = append(where, "driver = ?")
		args = append(args, filter.Driver)
	}
	if filter.Query != "" {
		where = append(where, `name LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(filter.Query)+"%")
	}
	if filter.Cursor != "" {
		where = append(where, "ref < ?")
		args = append(args, filter.Cursor)
	}

	query := `SELECT ` + targetColumns + ` FROM targets`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY ref DESC LIMIT ?"
	args = append(args, filter.Limit)

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("state: list targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	targets := make([]domain.Target, 0, filter.Limit)
	for rows.Next() {
		target, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("state: list targets: %w", err)
	}
	return targets, nil
}

// Update writes back a target that already exists. A missing row is reported as such: a
// silent zero would hide a lost write.
func (d *DB) Update(ctx context.Context, target domain.Target) (domain.Target, error) {
	options, err := encodeOptions(target.Options)
	if err != nil {
		return domain.Target{}, err
	}
	res, err := d.db.ExecContext(ctx, `UPDATE targets SET
			name = ?, host = ?, port = ?, database = ?, username = ?, password_cipher = ?,
			tls_mode = ?, options = ?, archived_at = ?, updated_at = ?
		WHERE ref = ?`,
		target.Name, target.Host, target.Port, target.Database, target.Username, target.Secret,
		target.TLSMode, options, nullableMillis(target.ArchivedAt), millis(target.UpdatedAt),
		target.Ref,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Target{}, fmt.Errorf("%w: a target named %q already exists", domain.ErrConflict, target.Name)
		}
		return domain.Target{}, fmt.Errorf("state: update target: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.Target{}, fmt.Errorf("state: update target: %w", err)
	}
	if affected == 0 {
		return domain.Target{}, fmt.Errorf("%w: target %s", domain.ErrNotFound, target.Ref)
	}
	return target, nil
}

// scanner is what both *sql.Row and *sql.Rows satisfy.
type scanner interface {
	Scan(dest ...any) error
}

func scanTarget(row scanner) (domain.Target, error) {
	var (
		target     domain.Target
		options    string
		archivedAt *int64
		createdAt  int64
		updatedAt  int64
	)
	err := row.Scan(&target.Ref, &target.Name, &target.Driver, &target.Host, &target.Port,
		&target.Database, &target.Username, &target.Secret, &target.TLSMode, &options,
		&target.CreatedBy, &archivedAt, &createdAt, &updatedAt)
	if err != nil {
		return domain.Target{}, err
	}
	if target.Options, err = decodeOptions(options); err != nil {
		return domain.Target{}, err
	}
	if archivedAt != nil {
		at := fromMillis(*archivedAt)
		target.ArchivedAt = &at
	}
	target.CreatedAt = fromMillis(createdAt)
	target.UpdatedAt = fromMillis(updatedAt)
	return target, nil
}

func encodeOptions(options map[string]string) (string, error) {
	if len(options) == 0 {
		return "{}", nil
	}
	buf, err := json.Marshal(options)
	if err != nil {
		return "", fmt.Errorf("state: encode options: %w", err)
	}
	return string(buf), nil
}

func decodeOptions(raw string) (map[string]string, error) {
	if raw == "" {
		return nil, nil
	}
	var options map[string]string
	if err := json.Unmarshal([]byte(raw), &options); err != nil {
		return nil, fmt.Errorf("state: decode options: %w", err)
	}
	return options, nil
}

// escapeLike keeps a typed % or _ literal instead of turning it into a wildcard.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
