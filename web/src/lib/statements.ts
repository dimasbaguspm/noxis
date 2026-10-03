// Pure statement classification, for UI hints only — the server is the
// authority (it guards and records the change kind).

export type StatementKind = "read" | "dml" | "ddl" | "empty";

const READ = new Set(["SELECT", "SHOW", "EXPLAIN", "DESCRIBE", "DESC"]);
const DDL = new Set(["ALTER", "CREATE", "DROP", "TRUNCATE", "RENAME", "GRANT", "REVOKE", "COMMENT"]);

function statements(sql: string): string[] {
  return sql
    .split(";")
    .map((s) => s.trim())
    .filter(Boolean);
}

function keyword(statement: string): string {
  return statement.split(/\s+/)[0]?.toUpperCase() ?? "";
}

// classify reports what the batch mostly is; any DDL wins (stricter review).
export function classify(sql: string): StatementKind {
  const parts = statements(sql);
  if (parts.length === 0) return "empty";
  if (parts.some((p) => DDL.has(keyword(p)))) return "ddl";
  if (parts.every((p) => READ.has(keyword(p)))) return "read";
  return "dml";
}

export function isReadOnly(sql: string): boolean {
  return classify(sql) === "read";
}
