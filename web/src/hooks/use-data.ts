import { useCallback, useEffect, useMemo, useState } from "react";
import { api, type Change, type ChangeInput, type Result } from "@/lib/api";
import { useAction } from "@/hooks/use-action";
import { classify } from "@/lib/statements";

export type ChangeKind = "dml" | "ddl";

// useChanges lists changes (optionally filtered by kind and by connection
// name) and exposes the workflow actions, reloading the list after each one.
export function useChanges(kind?: ChangeKind, connection?: string) {
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(async () => {
    try {
      const { changes: list } = await api.changes.list(connection);
      setChanges(list ?? []);
    } finally {
      setLoading(false);
    }
  }, [connection]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const submit = useAction(async (input: ChangeInput) => {
    const change = await api.changes.submit(input);
    await reload();
    return change;
  });
  const approve = useAction(async (id: string) => {
    await api.changes.approve(id);
    await reload();
  });
  const apply = useAction(async (id: string) => {
    await api.changes.apply(id);
    await reload();
  });
  const reject = useAction(async (id: string) => {
    await api.changes.reject(id);
    await reload();
  });

  // rows recorded before the server knew the kind fall back to the pure
  // classifier, so nothing disappears from both tabs
  const visible = useMemo(() => {
    if (!kind) return changes;
    return changes.filter((c) => (c.kind || classify(c.query ?? "")) === kind);
  }, [changes, kind]);

  return { changes: visible, all: changes, total: changes.length, loading, reload, submit, approve, apply, reject };
}

// useTableRows previews a table's rows for the active connection.
export function useTableRows(connection?: string) {
  const [result, setResult] = useState<Result | null>(null);
  const [label, setLabel] = useState("");

  const pick = useAction(async (schema: string, table: string) => {
    if (!connection) return;
    const rows = await api.connections.rows(connection, schema, table);
    setResult(rows);
    setLabel(`${schema}.${table}`);
  });

  return { result, label, pick: pick.run, loading: pick.loading };
}

// useQuery runs a read against the active connection.
export function useQuery(connection?: string) {
  const [result, setResult] = useState<Result | null>(null);

  const execute = useAction(async (sql: string) => {
    if (!connection) return;
    setResult(await api.ops.query(connection, sql));
  });

  return { result, execute: execute.run, loading: execute.loading };
}
