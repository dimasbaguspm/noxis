import { useMutation, useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { api, type MongoChangeInput, type MongoRequest } from "@/lib/api";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useUrlState } from "@/hooks/use-url-state";
import { useDebounce } from "@/hooks/use-debounce";
import { useChangeListUrlParams } from "@/hooks/use-changes";
import { queryClient, queryKeys } from "@/lib/query-client";

// The request queue lives under the driver in the URL, the same way the change queue does.

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
  void queryClient.invalidateQueries({ queryKey: queryKeys.requests.all });
}

export function useMongoChangesListQuery(scope?: { driver?: string; connection?: string }) {
  const { driver: routeDriver = "" } = useParams();
  const driver = scope?.driver ?? routeDriver;
  const { connections, name, status, page } = useChangeListUrlParams();
  const search = useDebounce(name.trim());
  const statusFilter = status === "all" ? undefined : status;
  // an instance's own page reviews one connection, so the picker is not consulted
  const targets = scope?.connection
    ? [scope.connection]
    : connections.length > 0
      ? connections
      : [undefined];
  const query = useQuery({
    queryKey: queryKeys.requests.list(driver, targets.join(","), search, statusFilter, page),
    // the filters run on the server before it paginates, exactly like the SQL queues
    queryFn: async () => {
      const pages = await Promise.all(
        targets.map((connection) =>
          api.mongodb(driver).changes.list(connection, page, statusFilter, search || undefined),
        ),
      );
      return {
        changes: pages.flatMap((result) => result.changes),
        total: pages.reduce((sum, result) => sum + result.total, 0),
      };
    },
    enabled: !!driver,
  });
  return { ...query, changes: query.data?.changes ?? [], total: query.data?.total };
}

export function useMongoChangeGetQuery(id?: string) {
  const { driver = "" } = useParams();
  return useQuery({
    queryKey: queryKeys.requests.get(driver, id ?? ""),
    queryFn: () => api.mongodb(driver).changes.get(id as string),
    enabled: !!id && !!driver,
  });
}

export function useMongoSubmitChangeMutation() {
  const { driver = "" } = useParams();
  return useMutation({
    mutationFn: (input: MongoChangeInput) => api.mongodb(driver).changes.submit(input),
    onSuccess: invalidateRequests,
  });
}

export function useMongoApproveChangeMutation() {
  const { driver = "" } = useParams();
  return useMutation({
    mutationFn: (id: string) => api.mongodb(driver).changes.approve(id),
    onSuccess: invalidateRequests,
  });
}

export function useMongoApplyChangeMutation() {
  const { driver = "" } = useParams();
  return useMutation({
    mutationFn: (id: string) => api.mongodb(driver).changes.apply(id),
    onSuccess: invalidateRequests,
  });
}

export function useMongoRejectChangeMutation() {
  const { driver = "" } = useParams();
  return useMutation({
    mutationFn: (id: string) => api.mongodb(driver).changes.reject(id),
    onSuccess: invalidateRequests,
  });
}
