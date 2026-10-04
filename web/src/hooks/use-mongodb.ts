import { useEffect, useState } from "react";
import { api, type MongoCollection, type MongoDatabase, type MongoStats, type Result } from "@/lib/api";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";

// useMongoConnections reads the MongoDB registry from the provider, so the
// dashboard and these pages never disagree.
export function useMongoConnections() {
  const { mongoConnections, mongoActive, setMongoActive, reloadMongo } = useApp();
  return {
    connections: mongoConnections,
    connection: mongoActive,
    pick: setMongoActive,
    reload: reloadMongo,
  };
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

// useMongoRequest runs one CRUD request and keeps its result.
export function useMongoRequest() {
  const [result, setResult] = useState<Result | null>(null);
  const run = useAction(async (name: string, body: Parameters<typeof api.mongodb.run>[1]) => {
    setResult(await api.mongodb.run(name, body));
  });
  return { result, run, loading: run.loading, reset: () => setResult(null) };
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
