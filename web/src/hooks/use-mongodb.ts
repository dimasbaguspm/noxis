import { useCallback, useEffect, useState } from "react";
import {
  api,
  type Change,
  type MongoChangeInput,
  type MongoCollection,
  type MongoDatabase,
  type MongoResult,
  type MongoStats,
} from "@/lib/api";
import { useFamily } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";

// useMongoConnections reads the MongoDB registry from the provider, so the
// dashboard and these pages never disagree.
export function useMongoConnections() {
  const { connections, active: connection, setActive: pick, reload } = useFamily("mongodb");
  return { connections, connection, pick, reload };
}

// useMongoDatabases lists the databases with their sizes.
export function useMongoDatabases(connection?: string) {
  const [databases, setDatabases] = useState<MongoDatabase[]>([]);
  const load = useAction(async (name: string) => {
    const res = await api.mongodb.databases(name);
    setDatabases(res.databases ?? []);
  });

  useEffect(() => {
    if (!connection) {
      setDatabases([]);
      return;
    }
    void load.run(connection);
  }, [connection, load.run]);

  return { databases, loading: load.loading, reload: () => connection && void load.run(connection) };
}

// useMongoCollections lists a database's collections with their sizes.
export function useMongoCollections(connection?: string, database?: string) {
  const [collections, setCollections] = useState<MongoCollection[]>([]);
  const load = useAction(async (name: string, db: string) => {
    const res = await api.mongodb.collections(name, db);
    setCollections(res.collections ?? []);
  });

  useEffect(() => {
    if (!connection || !database) {
      setCollections([]);
      return;
    }
    void load.run(connection, database);
  }, [connection, database, load.run]);

  return { collections, loading: load.loading, reload: () => connection && database && void load.run(connection, database) };
}

// useMongoRead runs one read request (find, aggregate, count) and keeps its
// result. Writes are not accepted here: they belong in the request queue.
export function useMongoRead() {
  const [result, setResult] = useState<MongoResult | null>(null);
  const run = useAction(async (name: string, body: Parameters<typeof api.mongodb.run>[1]) => {
    setResult(await api.mongodb.run(name, body));
  });
  return { result, run, loading: run.loading, reset: () => setResult(null) };
}

// useMongoChanges is the request queue: submit validates, approve signs off,
// apply runs (there is no rollback to fall back on).
export function useMongoChanges() {
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(true);

  const reload = useCallback(async () => {
    try {
      const { changes: list } = await api.mongodb.changes.list();
      setChanges(list ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  const submit = useAction(async (input: MongoChangeInput) => {
    const change = await api.mongodb.changes.submit(input);
    await reload();
    return change;
  });
  const approve = useAction(async (id: string) => {
    const change = await api.mongodb.changes.approve(id);
    await reload();
    return change;
  });
  const apply = useAction(async (id: string) => {
    const change = await api.mongodb.changes.apply(id);
    await reload();
    return change;
  });
  const reject = useAction(async (id: string) => {
    const change = await api.mongodb.changes.reject(id);
    await reload();
    return change;
  });

  useEffect(() => {
    void reload();
  }, [reload]);

  return { changes, loading, reload, submit, approve, apply, reject };
}

// useMongoStatsStream subscribes to the stats stream (2s ticks), scoped to a
// database and (optionally) a collection.
export function useMongoStatsStream(connection?: string, database?: string, collection?: string) {
  const [history, setHistory] = useState<MongoStats[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!connection) {
      setHistory([]);
      return;
    }
    setHistory([]);
    const source = new EventSource(api.mongodb.statsStream(connection, database, collection));
    source.addEventListener("open", () => setConnected(true));
    source.addEventListener("error", () => setConnected(false));
    source.addEventListener("stats", (event) => {
      try {
        const next = JSON.parse((event as MessageEvent).data) as MongoStats;
        setHistory((prev) => [...prev, next].slice(-60));
      } catch {
        /* ignore malformed frames */
      }
    });
    return () => {
      source.close();
      setConnected(false);
    };
  }, [connection, database, collection]);

  return { latest: history.at(-1), history, connected };
}
