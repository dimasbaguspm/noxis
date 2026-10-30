import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useConnections } from "@/hooks/use-connections";
import { useUrlState } from "@/hooks/use-url-state";
import { CAPABILITY } from "@/lib/console-nav";
import { queryClient, queryKeys } from "@/lib/query-client";

// useMemcacheConnections reads the memory-cache list from the cached instances query, so the dashboard and these pages never disagree. The active connection's driver comes along: redis and valkey are different engine paths.
const MEMCACHE_DEFAULTS = { pattern: "*", key: "" };

// useMemcacheUrlParams is the keys filter and the inspected key, kept in the URL.
export function useMemcacheUrlParams() {
  const [state, patch] = useUrlState(MEMCACHE_DEFAULTS);
  return {
    pattern: state.pattern,
    inspected: state.key || undefined,
    setPattern: (value: string) => patch({ pattern: value }),
    setInspected: (value: string | undefined) => patch({ key: value ?? "" }),
  };
}

export function useMemcacheConnections() {
  const { connections, active, driver, setActive, busy, reload } =
    useConnections(CAPABILITY.KEYS);
  return { connections, connection: active, driver, pick: setActive, busy, reload };
}

// useMemcacheKeysQuery browses keys by pattern (a bounded full scan server side); the page is cached per pattern so the debounced search replays fast.
export function useMemcacheKeysQuery(driver?: string, connection?: string, pattern = "*") {
  const query = useQuery({
    queryKey: queryKeys.keys.keys(connection, pattern),
    queryFn: () => api.memcache(driver as string).keys(connection as string, pattern),
    enabled: !!driver && !!connection,
  });
  return { ...query, page: query.data ?? { keys: [], total: 0, truncated: false, pattern } };
}

export function useMemcacheValueQuery(driver?: string, connection?: string, key?: string) {
  const query = useQuery({
    queryKey: queryKeys.keys.value(connection, key),
    queryFn: () => api.memcache(driver as string).value(connection as string, key as string),
    enabled: !!driver && !!connection && !!key,
  });
  return { ...query, result: query.data ?? null };
}

// Removing keys and emptying the database invalidate the keyspace cache; the page never calls refetch by hand after an action.
export function useRemoveKeysMutation() {
  return useMutation({
    mutationFn: ({
      driver,
      connection,
      keys,
    }: {
      driver: string;
      connection: string;
      keys: string[];
    }) => api.memcache(driver).removeKeys(connection, keys),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: queryKeys.keys.all }),
  });
}

export function useFlushMutation() {
  return useMutation({
    mutationFn: ({ driver, connection }: { driver: string; connection: string }) =>
      api.memcache(driver).flush(connection),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: queryKeys.keys.all }),
  });
}
