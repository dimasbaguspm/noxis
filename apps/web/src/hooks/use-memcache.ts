import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { useConnections } from "@/hooks/use-connections";
import { FAMILY } from "@/lib/families";
import { queryClient, queryKeys } from "@/lib/query-client";

// useMemcacheConnections reads the memory-cache list from the cached
// instances query, so the dashboard and these pages never disagree.
export function useMemcacheConnections() {
  const { connections, active, setActive, busy, reload } = useConnections(FAMILY.MEMCACHE);
  return { connections, connection: active, pick: setActive, busy, reload };
}

// useMemcacheKeysQuery browses keys by pattern (a bounded full scan server
// side); the page is cached per pattern so the debounced search replays fast.
export function useMemcacheKeysQuery(connection?: string, pattern = "*") {
  const query = useQuery({
    queryKey: queryKeys.memcache.keys(connection, pattern),
    queryFn: () => api.memcache.keys(connection as string, pattern),
    enabled: !!connection,
  });
  return { ...query, page: query.data ?? { keys: [], total: 0, truncated: false, pattern } };
}

export function useMemcacheValueQuery(connection?: string, key?: string) {
  const query = useQuery({
    queryKey: queryKeys.memcache.key(connection, key),
    queryFn: () => api.memcache.value(connection as string, key as string),
    enabled: !!connection && !!key,
  });
  return { ...query, result: query.data ?? null };
}

// Removing keys and emptying the database invalidate the keyspace cache; the
// page never calls refetch by hand after an action.
export function useRemoveKeysMutation() {
  return useMutation({
    mutationFn: ({ connection, keys }: { connection: string; keys: string[] }) =>
      api.memcache.removeKeys(connection, keys),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["memcache"] }),
  });
}

export function useFlushMutation() {
  return useMutation({
    mutationFn: (connection: string) => api.memcache.flush(connection),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["memcache"] }),
  });
}