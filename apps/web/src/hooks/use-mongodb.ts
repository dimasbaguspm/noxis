import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type MongoChangeInput, type MongoRequest } from "@/lib/api";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useUrlState } from "@/hooks/use-url-state";
import { queryClient, queryKeys } from "@/lib/query-client";

// MONGO_DRIVER is the driver whose request queue the console reads; the engine reads take the driver from the route.
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

// useMongoConnections is the documents page's instance list: the driver comes from the URL and the instance from ?instance=.
export function useMongoConnections() {
  return useDriverConnections();
}

export function useMongoDatabasesQuery(driver?: string, connection?: string) {
  const query = useQuery({
    queryKey: queryKeys.explore.databases(driver, connection),
    queryFn: () => api.mongodb(driver as string).databases(connection as string),
    enabled: !!driver && !!connection,
    select: (res) => res.databases,
  });
  return { ...query, databases: query.data ?? [] };
}

export function useMongoCollectionsQuery(driver?: string, connection?: string, database?: string) {
  const query = useQuery({
    queryKey: queryKeys.explore.collections(driver, connection, database),
    queryFn: () =>
      api.mongodb(driver as string).collections(connection as string, database as string),
    enabled: !!driver && !!connection && !!database,
    select: (res) => res.collections,
  });
  return { ...query, collections: query.data ?? [] };
}

// useMongoRunMutation executes one read (find, aggregate, count); writes are not accepted here and belong in the request queue.
export function useMongoRunMutation() {
  return useMutation({
    mutationFn: ({
      driver,
      connection,
      body,
    }: {
      driver: string;
      connection: string;
      body: MongoRequest;
    }) => api.mongodb(driver).run(connection, body),
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
