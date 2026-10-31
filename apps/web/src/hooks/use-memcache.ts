import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type MemcacheKey } from "@/lib/api";
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

// useMemcacheKeysQuery browses keys by pattern and carries the scan cursor, so a keyspace larger than one page can be
// walked past it. Paging is append-only on purpose: a SCAN cursor is not a snapshot, so the pages already shown stay put.
export function useMemcacheKeysQuery(driver?: string, connection?: string, pattern = "*") {
  const query = useQuery({
    queryKey: queryKeys.keys.keys(driver, connection, pattern),
    queryFn: () => api.memcache(driver as string).keys(connection as string, pattern),
    enabled: !!driver && !!connection,
  });
  const first = query.data;
  // held is the pages fetched after the first, tagged with the scope they belong to so a pattern change drops them.
  const scope = `${driver}|${connection}|${pattern}`;
  const [held, setHeld] = useState<{ scope: string; keys: MemcacheKey[]; cursor: number }>({
    scope: "",
    keys: [],
    cursor: 0,
  });
  const paged =
    held.scope === scope ? held : { keys: [] as MemcacheKey[], cursor: first?.cursor ?? 0 };
  const loadMore = useMutation({
    mutationFn: () =>
      api.memcache(driver as string).keys(connection as string, pattern, paged.cursor),
    onSuccess: (page) =>
      setHeld({
        scope,
        keys: [...paged.keys, ...(page.keys ?? [])],
        cursor: page.cursor ?? 0,
      }),
  });
  return {
    ...query,
    page: {
      keys: [...(first?.keys ?? []), ...paged.keys],
      total: first?.total ?? 0,
      truncated: paged.cursor !== 0,
      pattern,
    },
    loadMore,
  };
}

// useMemcacheExpireMutation sets or clears the expiry of the selected keys, then refreshes the page and the server's own
// numbers, because an expiry change moves the database's expires count too.
export function useMemcacheExpireMutation(driver?: string, connection?: string) {
  return useMutation({
    mutationFn: ({ keys, ttlMs }: { keys: string[]; ttlMs: number }) =>
      api.memcache(driver as string).expire(connection as string, keys, ttlMs),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.keys.all });
    },
  });
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
