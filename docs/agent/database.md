# Database, repositories & secrets

Detail for `AGENTS.md`.

## One store, one file

`apps/api/internal/repositories/sqlite.Store` is the **only** state. It opens a
`modernc.org/sqlite` (pure Go, cgo-free) database at `NOXIS_DB`, sets
`SetMaxOpenConns(1)` (SQLite writer serialization), applies the schema, then
the migrations, then backfills, then encrypts legacy passwords.

`Open()` guards the classic bind-mount trap: a missing bind target becomes a
**directory**, and sqlite cannot open a directory — it returns a clear error
instead of crash-looping.

## Schema (`apps/api/internal/repositories/sqlite/sqlite.go`, `schema` const)

### `connections`
| column | notes |
| --- | --- |
| `name` | PK, the connection's addressable name |
| `id` | stable instance id (`/external/v1/instances`), unique index, backfilled |
| `driver` | `postgres`/`mysql`/`redis`/`valkey`/`mongo` |
| `host`, `port`, `user`, `database` | identity fields, locked after registration |
| `password` | plaintext or `enc:v1:...` (AES-256-GCM) |
| `params` | JSON `map[string]string` extra DSN params |
| `created_at`, `created_by` | audit |

### `changes`
The reviewed-change artifact. Never deleted. `status` ∈
`validated|unverified|failed|approved|applying|applied|rejected`; `kind` ∈ `dml|ddl|mongo`.
`applying` is an atomic execution claim with `applying_at`; after a process crash,
the remote outcome is unknown, so the scheduler never retries it automatically.
Keep it visible for manual investigation; do not reset it without confirming the
target outcome. The row also carries checks, estimates and actuals, scratch schema,
affected identifiers, `scheduled_by`, `created_by`, `approved_by`, `rejected_by`, `applied_by`, and review/apply timestamps.

### `permissions`
`(connection, member, member_type, level)` PK `(connection, member)`. `member` is
a user subject (`local:<name>` or an OIDC sub) or a group name; `member_type` ∈
`user|group`; `level` ∈ `read|write|admin`. The only policy table. The old
`connection_grants` table is migrated in as `member_type='group'` and dropped.

### `users`
Accounts. `subject` PK (`local:<email>` or the OIDC sub), `email` is the blend
key: an OIDC login whose verified email equals an account's email signs in as
that account (unique when non-empty; login resolves by email or legacy
username), `username` (login name for a named local account, empty for OIDC
rows), `password` is a **bcrypt hash** (one-way, unlike `connections.password`),
`groups` JSON, `role` ∈ `super|admin|member`, `status` ∈ `active|banned`,
`last_login_at`, and the single active session: `token_hash` (SHA-256 of the raw
token) + `token_expires_at`. The raw token and password are never stored. A ban
clears the session; `UserByToken` only matches `status='active'`.

Exactly one row holds `role='super'`: the env root. Boot promotes it and demotes
any other `super` (`SetSoleSuper`); promoted admins are left alone.

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
  `connections.go`, `changes.go`, `permissions.go`, `users.go`.
- Shared row iteration lives in `queryRows[T]`; transactions via
  `Store.WithTx(ctx, fn)`.
- `Store` satisfies all three interfaces (`var _ interfaces.ConnectionRepository = (*Store)(nil)` etc.).
- Add a repository method → add it to the interface in `apps/api/internal/interfaces/repository.go`
  **and** to `Store`. Keep the `noun` in the method name because one Store
  implements several seams (`ListChanges`, `GetChange`, `SaveChange`).
- `constants.ErrNotFound` is the missing-row sentinel; handlers map it to 404.

## Secrets

`apps/api/internal/common/secret`:

- Passwords cannot be hashed (noxis must send them to the DB) → reversible
  encryption at rest.
- `NOXIS_SECRET_KEY`: 64 hex chars or 32 raw bytes. Required: the server refuses
to start without it unless `NOXIS_ALLOW_PLAINTEXT=true` (local dev only).
- Ciphertext carries the `enc:v1:` prefix, so plaintext rows written before the
  key existed stay readable.
- `Store.encryptLegacyPasswords` re-seals plaintext rows when a key is present
  (convergence on boot).
- `Connection.Redacted()` blanks the password on every served shape. Never
  return a raw `Connection` to the wire.

## Limits & config

`apps/api/internal/constants/limits.go`: `DefaultPageSize=20`, `MaxPageSize=100`,
`ScanLimit=500`, `MaxScanLimit=5000`. Paging is `apps/api/internal/common/pagination`
(pure functions, no `net/http`).

## What is *not* in the DB

Runtime engines are separate servers noxis connects to — noxis does not
provision them. There is no desired-state/status oplog, no audit table beyond the
`changes` artifact, and no user table (identity is OIDC).
