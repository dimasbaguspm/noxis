import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type MongoChangeInput, type MongoRequest } from "@/lib/api";
import { useConnections } from "@/hooks/use-connections";
import { useUrlState } from "@/hooks/use-url-state";
import { CAPABILITY } from "@/lib/console-nav";
import { queryClient, queryKeys } from "@/lib/query-client";

// Every call here is addressed by the MongoDB driver, which is one name in the server's catalog.
const MONGO_DRIVER = "mongo";

// useMongoConnections reads the document connections from the cached instances query, so the dashboard and these pages never disagree.
export type MongoView = "json" | "raw" | "tree";
const MONGO_DEFAULTS = { database: "", collection: "", filter: "{}", view: "json" };

// useMongoUrlParams keeps selection in the URL and clears filters when the database or collection changes.
export function useMongoUrlParams() {
  const [state, patch] = useUrlState(MONGO_DEFAULTS);
  return {
    database: state.database || undefined,
    collection: state.collection || undefined,
    filter: state.filter,
    view: state.view as MongoView,
    setDatabase: (value?: string) => patch({ database: value ?? "", collection: "", filter: "{}" }),
    setCollection: (value?: string) => patch({ collection: value ?? "", filter: "{}" }),
    setFilter: (value: string) => patch({ filter: value }),
    setView: (value: MongoView) => patch({ view: value }),
  };
}

export function useMongoConnections() {
  const { connections, active, setActive, busy, reload } = useConnections(CAPABILITY.DOCUMENTS);
  return { connections, connection: active, pick: setActive, busy, reload };
}

export function useMongoDatabasesQuery(connection?: string) {
  const query = useQuery({
    queryKey: queryKeys.explore.databases(connection),
    queryFn: () => api.mongodb(MONGO_DRIVER).databases(connection as string),
    enabled: !!connection,
    select: (res) => res.databases,
  });
  return { ...query, databases: query.data ?? [] };
}

export function useMongoCollectionsQuery(connection?: string, database?: string) {
  const query = useQuery({
    queryKey: queryKeys.explore.collections(connection, database),
    queryFn: () => api.mongodb(MONGO_DRIVER).collections(connection as string, database as string),
    enabled: !!connection && !!database,
    select: (res) => res.collections,
  });
  return { ...query, collections: query.data ?? [] };
}

// useMongoRunMutation executes one read (find, aggregate, count); writes are not accepted here and belong in the request queue.
export function useMongoRunMutation() {
  return useMutation({
    mutationFn: ({ connection, body }: { connection: string; body: MongoRequest }) =>
      api.mongodb(MONGO_DRIVER).run(connection, body),
  });
}

// Request queue: list, detail and the review actions. Every action invalidates the queue so the list and detail refetch by themselves.
function invalidateRequests() {
  void queryClient.invalidateQueries({ queryKey: queryKeys.requests.list() });
}

export function useMongoChangesListQuery() {
  const query = useQuery({
    queryKey: queryKeys.requests.list(),
    queryFn: () => api.mongodb(MONGO_DRIVER).changes.list(),
  });
  return { ...query, changes: query.data?.changes ?? [] };
}

export function useMongoChangeGetQuery(id?: string) {
  return useQuery({
    queryKey: queryKeys.requests.get(id ?? ""),
    queryFn: () => api.mongodb(MONGO_DRIVER).changes.get(id as string),
    enabled: !!id,
  });
}

export function useMongoSubmitChangeMutation() {
  return useMutation({
    mutationFn: (input: MongoChangeInput) => api.mongodb(MONGO_DRIVER).changes.submit(input),
    onSuccess: invalidateRequests,
  });
}

export function useMongoApproveChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.mongodb(MONGO_DRIVER).changes.approve(id),
    onSuccess: invalidateRequests,
  });
}

export function useMongoApplyChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.mongodb(MONGO_DRIVER).changes.apply(id),
    onSuccess: invalidateRequests,
  });
}

export function useMongoRejectChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.mongodb(MONGO_DRIVER).changes.reject(id),
    onSuccess: invalidateRequests,
  });
}
