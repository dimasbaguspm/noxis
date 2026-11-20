# The state file

One SQLite file, written by `noxisd` and by nothing else. It holds targets, the journal,
and — as later tickets land — proposals and rehearsals. It is an audit trail before it is
a cache, which is why nothing here migrates one.

## Lifecycle

`internal/adapters/sqlite.Open`:

1. Creates the parent directory when it is missing.
2. Opens the file with `journal_mode(WAL)`, `busy_timeout(5000)`, `foreign_keys(1)`,
   `synchronous(NORMAL)`, and **one** connection: SQLite serialises writes anyway, and a
   single writer is what makes a claim-then-act pattern behind `apply` honest.
3. Reads `PRAGMA user_version`:
   - equal to `SchemaVersion` → ready;
   - zero and the file holds no tables → create the schema and stamp the version;
   - zero but the file holds tables → refuse: it is not a state file this build wrote;
   - anything else → refuse with the version it found and the one this build speaks.
4. A refusal names the path and says to start with a fresh file. There is no downgrade
   path and no migration, because silently dropping a journal is worse than a failed boot.

`SchemaVersion` lives next to the schema in `internal/adapters/sqlite/open.go`. A ticket
that changes the schema bumps it.

## Tables (schema v4)

An older file is refused rather than migrated, like any other version: the reset is the
migration.

**targets** — the servers this build governs.

| column | type | note |
| --- | --- | --- |
| `ref` | TEXT | primary key, UUIDv7 |
| `name` | TEXT | unique, 2–63 chars of `a-z0-9-_` |
| `driver` | TEXT | `mysql` or `mariadb` |
| `host`, `database`, `username` | TEXT | as typed by the operator |
| `port` | INTEGER | default 3306 |
| `password_cipher` | TEXT | AES-256-GCM envelope `v1:<b64>`, AAD = `ref`; empty when there is no password |
| `tls_mode` | TEXT | `preferred`, `skip-verify`, `verify` |
| `options` | TEXT | JSON object of driver options |
| `created_by` | TEXT | the token name that registered it |
| `archived_at` | INTEGER | milliseconds, NULL while live |
| `server_version` | TEXT | what the last successful probe measured, empty until then |
| `last_probe_at` | INTEGER | milliseconds of the last attempt, successful or not |
| `last_probe_status` | TEXT | `ok` or `failed`, empty before the first attempt |
| `created_at`, `updated_at` | INTEGER | milliseconds |

**journal** — append-only. Nothing updates or deletes a row.

| column | type | note |
| --- | --- | --- |
| `ref` | TEXT | primary key, UUIDv7 |
| `subject_ref`, `subject_kind` | TEXT | `target` today, `proposal` later |
| `kind` | TEXT | `created`, `updated`, `archived`, `probed` |
| `actor` | TEXT | the token name |
| `detail` | TEXT | JSON object; never a secret |
| `created_at` | INTEGER | milliseconds |

Both tables are `STRICT`, so a wrong type is an error at write time rather than a surprise
at read time. Timestamps are integers, so ordering never depends on a string format.

**proposals** — one proposed change, rehearsed before a human sees it. `status` is written only by
the state machine in `internal/domain`; `tier` records what was actually run, so a reviewer never
assumes more proof than there is; `irreversible` is set from the rehearsal, not from the statement's
wording.

| column | type | note |
| --- | --- | --- |
| `started_at` | INTEGER | milliseconds, set when apply claimed the proposal |
| `progress_stage`, `progress_percent` | TEXT, INTEGER | what the engine said while it ran; percent is `-1` when the engine does not say |
| `algorithm` | TEXT | `INSTANT`, `INPLACE`, `COPY`, empty when the engine would not name one |
| `rows_estimated`, `rows_affected` | INTEGER | the estimate that decided whether the answer waits, and what the statement actually touched |

`progress_*` is written by `SetProgress`, which touches those two columns only: the watcher must
never overwrite the receipt that ends the apply. A row still `applying` after a restart is
indeterminate on purpose — nothing retries it.

**rehearsals** — every attempt, including the failures. `schema_diff` and `row_diff` are the
evidence a reviewer reads; `note` says exactly how far the claim goes (a shadow run with no rows
copied proves shape, not effect).

## Why a password is never in a payload

`password_cipher` is sealed with `SECRET_KEY` and bound to the row's `ref` as additional
authenticated data: a sealed value copied into another row will not open. The domain type
carries it as an opaque `Secret` string, the use case is the only place a plaintext
password exists, and the wire type in `cmd/http/handlers` has no password field at all —
only `has_password`. `SECRET_KEY_PREVIOUS` opens rows sealed with the old key and reports
them as stale so the next write re-seals them.

## Reading and paging

Every list orders by `ref DESC`. Because a UUIDv7 begins with its creation timestamp, that
column is both "newest first" and a stable keyset: a page is `WHERE ref < :cursor LIMIT
:limit+1`, and the extra row is how "is there another page" is answered without counting.
No query in this tree uses `OFFSET`.
