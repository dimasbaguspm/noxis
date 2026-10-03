import { useCallback, useEffect, useState } from "react";
import { api, type Connection, type RedisKeyPage, type RedisStats, type Result } from "@/lib/api";
import { useAction } from "@/hooks/use-action";

// Redis keeps its own connection list: the family is separate from the SQL
// instances, so it does not ride the app provider's shared state.
export function useRedisConnections() {
  const [connections, setConnections] = useState<Connection[]>([]);
  const [connection, setConnection] = useState<string>();

  const reload = useCallback(async () => {
    const res = await api.redis.list();
    setConnections(res.connections ?? []);
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    if (!connection && connections.length > 0) setConnection(connections[0]?.name);
  }, [connection, connections]);

  return { connections, connection, pick: setConnection, reload };
}

// useRedisKeys browses keys by pattern. The API does a bounded full scan, so
// the whole page can be sorted by TTL in the browser.
export function useRedisKeys(connection?: string, pattern = "*") {
  const [page, setPage] = useState<RedisKeyPage>({ keys: [], total: 0, truncated: false, pattern: "*" });
  const load = useAction(async (name: string, glob: string) => {
    setPage(await api.redis.keys(name, glob));
  });

  const reload = useCallback(() => {
    if (connection) void load.run(connection, pattern);
  }, [connection, pattern, load.run]);

  useEffect(() => {
    if (!connection) {
      setPage({ keys: [], total: 0, truncated: false, pattern: "*" });
      return;
    }
    void load.run(connection, pattern);
  }, [connection, pattern, load.run]);

  return { page, loading: load.loading, reload };
}

// useRedisValue renders one key's value.
export function useRedisValue(connection?: string, key?: string) {
  const [result, setResult] = useState<Result | null>(null);
  const load = useAction(async (name: string, target: string) => {
    setResult(await api.redis.value(name, target));
  });

  useEffect(() => {
    if (!connection || !key) {
      setResult(null);
      return;
    }
    void load.run(connection, key);
  }, [connection, key, load.run]);

  return { result, loading: load.loading };
}

// useRedisStatsStream subscribes to the INFO stream (2s ticks).
export function useRedisStatsStream(connection?: string) {
  const [history, setHistory] = useState<RedisStats[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!connection) {
      setHistory([]);
      return;
    }
    setHistory([]);
    const source = new EventSource(api.redis.statsStream(connection));
    source.addEventListener("open", () => setConnected(true));
    source.addEventListener("error", () => setConnected(false));
    source.addEventListener("stats", (event) => {
      try {
        const next = JSON.parse((event as MessageEvent).data) as RedisStats;
        setHistory((prev) => [...prev, next].slice(-60));
      } catch {
        /* ignore malformed frames */
      }
    });
    return () => {
      source.close();
      setConnected(false);
    };
  }, [connection]);

  return { latest: history.at(-1), history, connected };
}
