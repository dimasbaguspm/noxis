CREATE TABLE changes (
  id          TEXT PRIMARY KEY CHECK (length(id) > 0),
  connection  TEXT NOT NULL CHECK (length(connection) > 0),
  notes       TEXT NOT NULL CHECK (length(notes) > 0),
  query       TEXT NOT NULL DEFAULT '',
  rollback    TEXT NOT NULL DEFAULT '',
  kind        TEXT NOT NULL CHECK (kind IN (%s)),
  status      TEXT NOT NULL CHECK (status IN (%s)),
  checks      TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(checks)),
  unverified_reason TEXT NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL CHECK (created_at > 0),
  created_by  TEXT NOT NULL DEFAULT '',
  approved_at INTEGER NOT NULL DEFAULT 0 CHECK (approved_at >= 0),
  approved_by TEXT NOT NULL DEFAULT '',
  applying_at INTEGER NOT NULL DEFAULT 0 CHECK (applying_at >= 0),
  applied_at  INTEGER NOT NULL DEFAULT 0 CHECK (applied_at >= 0),
  applied_by  TEXT NOT NULL DEFAULT '',
  rejected_at INTEGER NOT NULL DEFAULT 0 CHECK (rejected_at >= 0),
  rejected_by TEXT NOT NULL DEFAULT '',
  est_duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (est_duration_ms >= 0),
  est_affected INTEGER NOT NULL DEFAULT 0 CHECK (est_affected >= 0),
  applied_affected INTEGER NOT NULL DEFAULT 0 CHECK (applied_affected >= 0),
  applied_duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (applied_duration_ms >= 0),
  scheduled_at INTEGER NOT NULL DEFAULT 0 CHECK (scheduled_at >= 0),
  scheduled_by TEXT NOT NULL DEFAULT '',
  affected_tables TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(affected_tables)),
  affected_columns TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(affected_columns))
) STRICT;

CREATE TABLE connections (
  name       TEXT PRIMARY KEY CHECK (length(trim(name)) > 0),
  id         TEXT NOT NULL CHECK (length(id) > 0),
  driver     TEXT NOT NULL CHECK (length(driver) > 0),
  host       TEXT NOT NULL DEFAULT '',
  port       INTEGER NOT NULL DEFAULT 0 CHECK (port BETWEEN 0 AND 65535),
  user       TEXT NOT NULL DEFAULT '',
  password   TEXT NOT NULL DEFAULT '',
  database   TEXT NOT NULL CHECK (length(database) > 0),
  params     TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(params)),
  created_at INTEGER NOT NULL CHECK (created_at > 0),
  created_by TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE TABLE permissions (
  connection  TEXT NOT NULL CHECK (length(connection) > 0),
  member      TEXT NOT NULL CHECK (length(member) > 0),
  member_type TEXT NOT NULL CHECK (member_type IN (%s)),
  level       TEXT NOT NULL CHECK (level IN (%s)),
  expires_at  INTEGER NOT NULL CHECK (expires_at > 0),
  created_by  TEXT NOT NULL,
  created_at  INTEGER NOT NULL CHECK (created_at > 0),
  updated_at  INTEGER NOT NULL CHECK (updated_at > 0),
  PRIMARY KEY (connection, member)
) STRICT;

CREATE TABLE users (
  subject          TEXT PRIMARY KEY CHECK (length(trim(subject)) > 0),
  email            TEXT NOT NULL DEFAULT '',
  username         TEXT NOT NULL DEFAULT '',
  password         TEXT NOT NULL DEFAULT '',
  groups           TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(groups)),
  role             TEXT NOT NULL CHECK (role IN (%s)),
  status           TEXT NOT NULL CHECK (status IN (%s)),
  token_hash       TEXT NOT NULL DEFAULT '',
  token_expires_at INTEGER NOT NULL DEFAULT 0 CHECK (token_expires_at >= 0),
  last_login_at    INTEGER NOT NULL DEFAULT 0 CHECK (last_login_at >= 0),
  created_at       INTEGER NOT NULL CHECK (created_at > 0),
  updated_at       INTEGER NOT NULL CHECK (updated_at > 0)
) STRICT;

CREATE TABLE access_requests (
  id            TEXT PRIMARY KEY CHECK (length(id) > 0),
  subject       TEXT NOT NULL CHECK (length(subject) > 0),
  connection    TEXT NOT NULL CHECK (length(connection) > 0),
  level         TEXT NOT NULL CHECK (level IN (%s)),
  reason        TEXT NOT NULL DEFAULT '',
  status        TEXT NOT NULL CHECK (status IN (%s)),
  requested_at  INTEGER NOT NULL CHECK (requested_at > 0),
  decided_by    TEXT NOT NULL DEFAULT '',
  decided_at    INTEGER NOT NULL DEFAULT 0 CHECK (decided_at >= 0),
  decision_note TEXT NOT NULL DEFAULT '',
  expires_at    INTEGER NOT NULL DEFAULT 0 CHECK (expires_at >= 0)
) STRICT;

CREATE TABLE access_events (
  id         TEXT PRIMARY KEY CHECK (length(id) > 0),
  at         INTEGER NOT NULL CHECK (at > 0),
  actor      TEXT NOT NULL,
  action     TEXT NOT NULL CHECK (length(action) > 0),
  subject    TEXT NOT NULL DEFAULT '',
  connection TEXT NOT NULL DEFAULT '',
  level      TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT ''
) STRICT;

CREATE UNIQUE INDEX connections_id ON connections(id);
CREATE UNIQUE INDEX users_email ON users(email) WHERE email != '';
CREATE UNIQUE INDEX users_username ON users(username) WHERE username != '';
CREATE UNIQUE INDEX users_token_hash ON users(token_hash) WHERE token_hash != '';
CREATE UNIQUE INDEX access_requests_pending ON access_requests(subject, connection) WHERE status = %s;
CREATE INDEX changes_by_created ON changes(created_at DESC, id);
CREATE INDEX changes_by_connection_created ON changes(connection, created_at DESC, id);
CREATE INDEX changes_by_schedule ON changes(status, scheduled_at) WHERE scheduled_at > 0;
CREATE INDEX permissions_by_member ON permissions(member, connection);
CREATE INDEX permissions_groups ON permissions(member_type, member);
CREATE INDEX access_requests_by_subject ON access_requests(subject, requested_at DESC);
CREATE INDEX access_requests_by_status_connection ON access_requests(status, connection, requested_at);
CREATE INDEX access_events_by_subject ON access_events(subject, at DESC);
CREATE INDEX access_events_by_actor ON access_events(actor, at DESC);
CREATE INDEX access_events_by_connection ON access_events(connection, at DESC);
