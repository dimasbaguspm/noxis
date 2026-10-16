# Database, repositories & secrets

Detail for `AGENTS.md`.

## One store, one file

`internal/repositories/sqlite.Store` is the **only** state. It opens a
`modernc.org/sqlite` (pure Go, cgo-free) database at `NOXIS_DB`, sets
`SetMaxOpenConns(1)` (SQLite writer serialization), applies the schema, then
the migrations, then backfills, then encrypts legacy passwords.

`Open()` guards the classic bind-mount trap: a missing bind target becomes a
**directory**, and sqlite cannot open a directory — it returns a clear error
instead of crash-looping.

## Schema (`internal/repositories/sqlite/sqlite.go`, `schema` const)

### `connections`
| column | notes |
| --- | --- |
| `name` | PK, the connection's addressable name |
| `id` | stable instance id (`/v1/instances`), unique index, backfilled |
| `driver` | `postgres`/`mysql`/`redis`/`valkey`/`mongo` |
| `host`, `port`, `user`, `database` | identity fields, locked after registration |
| `password` | plaintext or `enc:v1:...` (AES-256-GCM) |
| `params` | JSON `map[string]string` extra DSN params |
| `created_at`, `created_by` | audit |

### `changes`
The reviewed-change artifact. Never deleted. `status` ∈
`validated|unverified|failed|approved|applied|rejected`; `kind` ∈ `dml|ddl|mongo`.
Carries `checks` (JSON), `est_duration_ms`/`est_affected` (dry-run estimates),
`applied_affected`/`applied_duration_ms` (actuals), `scratch_schema`,
`affected_tables`/`affected_columns` (JSON), `scheduled_at`, `created_by`, and
the approve/apply/reject timestamps.

### `connection_grants`
`(connection, group_name, level)` PK, `level` ∈ `read|write|admin`. The only
policy table. Identity/membership live in the OIDC issuer.

## Migrations

- Append-only slice `migrations []string` of `ALTER TABLE ... ADD COLUMN`.
- Applied best-effort; duplicates (fresh DBs) error and are ignored.
- Never edit an existing line; add a new one. SQLite has no transactional DDL
  rollback here, so columns must be nullable or have a `DEFAULT`.
- Index creation that depends on backfill runs **after** the backfill
  (`connections_id` unique index), because an index over an empty default would
  collide.

## Repositories (the pattern)

- Each repository method is a `Store` method in its own file:
  `connections.go`, `changes.go`, `grants.go`.
- Shared row iteration lives in `queryRows[T]`; transactions via
  `Store.WithTx(ctx, fn)`.
- `Store` satisfies all three interfaces (`var _ interfaces.ConnectionRepository = (*Store)(nil)` etc.).
- Add a repository method → add it to the interface in `internal/interfaces/repository.go`
  **and** to `Store`. Keep the `noun` in the method name because one Store
  implements several seams (`ListChanges`, `GetChange`, `SaveChange`).
- `interfaces.ErrNotFound` is the missing-row sentinel; handlers map it to 404.

## Secrets

`internal/common/secret`:

- Passwords cannot be hashed (noxis must send them to the DB) → reversible
  encryption at rest.
- `NOXIS_SECRET_KEY`: 64 hex chars or 32 raw bytes. Empty = plaintext mode and a
  boot warning.
- Ciphertext carries the `enc:v1:` prefix, so plaintext rows written before the
  key existed stay readable.
- `Store.encryptLegacyPasswords` re-seals plaintext rows when a key is present
  (convergence on boot).
- `Connection.Redacted()` blanks the password on every served shape. Never
  return a raw `Connection` to the wire.

## Limits & config

`internal/constants/limits.go`: `DefaultPageSize=20`, `MaxPageSize=100`,
`ScanLimit=500`, `MaxScanLimit=5000`. Paging is `internal/common/pagination`
(pure functions, no `net/http`).

## What is *not* in the DB

Runtime engines are separate servers noxis connects to — noxis does not
provision them. There is no desired-state/status oplog, no audit table beyond the
`changes` artifact, and no user table (identity is OIDC).
