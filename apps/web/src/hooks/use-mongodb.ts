import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type MongoChangeInput } from "@/lib/api";
import { useConnections } from "@/hooks/use-connections";
import { useUrlState } from "@/hooks/use-url-state";
import { FAMILY } from "@/lib/families";
import { queryClient, queryKeys } from "@/lib/query-client";

// useMongoConnections reads the MongoDB list from the cached instances query,
// so the dashboard and these pages never disagree.
export type MongoView = "json" | "raw" | "tree";
const MONGO_DEFAULTS = { database: "", collection: "", filter: "{}", view: "json" };

// useMongoUrlParams is the explore page's selection, kept in the URL: database,
// collection, filter and view. Changing the database or collection resets the
// filter, so a new target starts from an empty read.
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
  const { connections, active, setActive, busy, reload } = useConnections(FAMILY.MONGODB);
  return { connections, connection: active, pick: setActive, busy, reload };
}

export function useMongoDatabasesQuery(connection?: string) {
  const query = useQuery({
    queryKey: queryKeys.mongodb.databases(connection),
    queryFn: () => api.mongodb.databases(connection as string),
    enabled: !!connection,
    select: (res) => res.databases,
  });
  return { ...query, databases: query.data ?? [] };
}

export function useMongoCollectionsQuery(connection?: string, database?: string) {
  const query = useQuery({
    queryKey: queryKeys.mongodb.collections(connection, database),
    queryFn: () => api.mongodb.collections(connection as string, database as string),
    enabled: !!connection && !!database,
    select: (res) => res.collections,
  });
  return { ...query, collections: query.data ?? [] };
}

// useMongoRunMutation executes one read (find, aggregate, count); writes are
// not accepted here and belong in the request queue.
export function useMongoRunMutation() {
  return useMutation({
    mutationFn: ({ connection, body }: { connection: string; body: Parameters<typeof api.mongodb.run>[1] }) =>
      api.mongodb.run(connection, body),
  });
}

// Request queue: list, detail and the review actions. Every action invalidates
// the queue so the list and detail refetch by themselves.
function invalidateRequests() {
  void queryClient.invalidateQueries({ queryKey: queryKeys.mongodb.requests.list() });
}

export function useMongoChangesListQuery() {
  const query = useQuery({
    queryKey: queryKeys.mongodb.requests.list(),
    queryFn: () => api.mongodb.changes.list(),
  });
  return { ...query, changes: query.data?.changes ?? [] };
}

export function useMongoChangeGetQuery(id?: string) {
  return useQuery({
    queryKey: queryKeys.mongodb.requests.get(id ?? ""),
    queryFn: () => api.mongodb.changes.get(id as string),
    enabled: !!id,
  });
}

export function useMongoSubmitChangeMutation() {
  return useMutation({
    mutationFn: (input: MongoChangeInput) => api.mongodb.changes.submit(input),
    onSuccess: invalidateRequests,
  });
}

export function useMongoApproveChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.mongodb.changes.approve(id),
    onSuccess: invalidateRequests,
  });
}

export function useMongoApplyChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.mongodb.changes.apply(id),
    onSuccess: invalidateRequests,
  });
}

export function useMongoRejectChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.mongodb.changes.reject(id),
    onSuccess: invalidateRequests,
  });
}