import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useUrlState } from "@/hooks/use-url-state";
import { queryClient, queryKeys } from "@/lib/query-client";

// MEMCACHE_DEFAULTS keeps the keys filter stable across renders.
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

// useMemcacheConnections is the keys page's instance list: the driver comes from the URL, so redis and valkey stay
// separate engine paths, and the instance comes from ?instance=.
export function useMemcacheConnections() {
  return useDriverConnections();
}

// useMemcacheKeysQuery browses keys by pattern (a bounded full scan server side); the page is cached per pattern so the debounced search replays fast.
export function useMemcacheKeysQuery(driver?: string, connection?: string, pattern = "*") {
  const query = useQuery({
    queryKey: queryKeys.keys.keys(driver, connection, pattern),
    queryFn: () => api.memcache(driver as string).keys(connection as string, pattern),
    enabled: !!driver && !!connection,
  });
  return { ...query, page: query.data ?? { keys: [], total: 0, truncated: false, pattern } };
}

// useMemcacheInfoQuery reads the server's own numbers; they change constantly, so they are refreshed on a slow poll
// rather than on every key action.
export function useMemcacheInfoQuery(driver?: string, connection?: string) {
  const query = useQuery({
    queryKey: queryKeys.keys.stats(driver, connection),
    queryFn: () => api.memcache(driver as string).stats(connection as string),
    enabled: !!driver && !!connection,
    refetchInterval: 30_000,
  });
  return { ...query, info: query.data };
}

export function useMemcacheValueQuery(driver?: string, connection?: string, key?: string) {
  const query = useQuery({
    queryKey: queryKeys.keys.value(driver, connection, key),
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
