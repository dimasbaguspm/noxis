import { useCallback, useEffect, useState } from "react";
import { api, type MemcacheKeyPage, type MemcacheStats, type Result } from "@/lib/api";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";

// useMemcacheConnections reads the memory-cache registry from the provider:
// the same list the dashboard shows, so the two never disagree.
export function useMemcacheConnections() {
  const { memcacheConnections, memcacheActive, setMemcacheActive, reloadMemcache } = useApp();
  return {
    connections: memcacheConnections,
    connection: memcacheActive,
    pick: setMemcacheActive,
    reload: reloadMemcache,
  };
}

// useMemcacheKeys browses keys by pattern. The API does a bounded full scan, so
// the whole page can be sorted by TTL in the browser.
export function useMemcacheKeys(connection?: string, pattern = "*") {
  const [page, setPage] = useState<MemcacheKeyPage>({ keys: [], total: 0, truncated: false, pattern: "*" });
  const load = useAction(async (name: string, glob: string) => {
    setPage(await api.memcache.keys(name, glob));
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

// useMemcacheValue renders one key's value.
export function useMemcacheValue(connection?: string, key?: string) {
  const [result, setResult] = useState<Result | null>(null);
  const load = useAction(async (name: string, target: string) => {
    setResult(await api.memcache.value(name, target));
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

// useMemcacheStatsStream subscribes to the INFO stream (2s ticks).
export function useMemcacheStatsStream(connection?: string) {
  const [history, setHistory] = useState<MemcacheStats[]>([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!connection) {
      setHistory([]);
      return;
    }
    setHistory([]);
    const source = new EventSource(api.memcache.statsStream(connection));
    source.addEventListener("open", () => setConnected(true));
    source.addEventListener("error", () => setConnected(false));
    source.addEventListener("stats", (event) => {
      try {
        const next = JSON.parse((event as MessageEvent).data) as MemcacheStats;
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
