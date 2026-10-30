import { useParams } from "react-router-dom";
import { useDebounce } from "@/hooks/use-debounce";
import { useUrlState } from "@/hooks/use-url-state";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type ChangeInput } from "@/lib/api";
import { queryClient, queryKeys } from "@/lib/query-client";

export type ChangeKind = "dml" | "ddl";

// The queue lives under the driver in the URL: the server scopes it to that driver, so no reader has to guess one.

// ./changes refetches the queue and any detail after an action, so a page never calls refetch() by hand.
function invalidateChanges() {
  void queryClient.invalidateQueries({ queryKey: queryKeys.changes.all });
}

const CHANGE_DEFAULTS = { q: "", name: "", status: "all", page: "1" };

// useChangeListUrlParams is the change queue's filter state, kept in the URL: the chosen instances are a comma list so a
// reload restores the same queue.
export function useChangeListUrlParams() {
  const [state, patch] = useUrlState(CHANGE_DEFAULTS);
  const connections = state.q ? state.q.split(",").filter(Boolean) : [];
  return {
    connections,
    name: state.name,
    status: state.status,
    page: Number(state.page),
    setConnections: (value: string[]) => patch({ q: value.join(","), page: "1" }),
    setName: (value: string) => patch({ name: value, page: "1" }),
    setStatus: (value: string) => patch({ status: value, page: "1" }),
    setPage: (value: number) => patch({ page: String(value) }),
  };
}

// useChangeListQuery asks the server once per chosen instance and merges the pages, so the total is the sum of the queues
// the reader actually asked for. A scoped queue (an instance's own page) has one instance and one driver, so the URL
// picker is not consulted at all.
export function useChangeListQuery(
  kind?: ChangeKind,
  scope?: { driver?: string; connection?: string },
) {
  const { driver: routeDriver = "" } = useParams();
  const driver = scope?.driver ?? routeDriver;
  const { connections, name, status, page } = useChangeListUrlParams();
  const search = useDebounce(name.trim());
  const statusFilter = status === "all" ? undefined : status;
  const targets = scope?.connection
    ? [scope.connection]
    : connections.length > 0
      ? connections
      : [undefined];
  const query = useQuery({
    queryKey: queryKeys.changes.list(
      driver,
      kind,
      connections.join(","),
      search,
      statusFilter,
      page,
    ),
    // the server filters by kind before it paginates, so each total is its own queue's count
    queryFn: async () => {
      const pages = await Promise.all(
        targets.map((connection) =>
          api
            .changes(driver)
            .list(connection, page, undefined, statusFilter, kind, search || undefined),
        ),
      );
      return {
        changes: pages.flatMap((result) => result.changes),
        total: pages.reduce((sum, result) => sum + result.total, 0),
      };
    },
    enabled: !!driver,
  });

  return { ...query, changes: query.data?.changes, total: query.data?.total };
}

// useChangeGetQuery reads one change through the driver in the URL, which also refuses a change from another driver.
export function useChangeGetQuery(id?: string) {
  const { driver = "" } = useParams();
  return useQuery({
    queryKey: queryKeys.changes.get(driver, id ?? ""),
    queryFn: () => api.changes(driver).get(id as string),
    enabled: !!id && !!driver,
  });
}

// Every mutation names the driver of the change's connection: submitting validates against it and applying runs through it.
export function useSubmitChangeMutation() {
  return useMutation({
    mutationFn: ({ driver, ...input }: ChangeInput & { driver: string }) =>
      api.changes(driver).submit(input),
    onSuccess: invalidateChanges,
  });
}

export function useApproveChangeMutation() {
  return useMutation({
    mutationFn: ({ id, driver }: { id: string; driver: string }) => api.changes(driver).approve(id),
    onSuccess: invalidateChanges,
  });
}

export function useApplyChangeMutation() {
  return useMutation({
    mutationFn: ({ id, driver }: { id: string; driver: string }) => api.changes(driver).apply(id),
    onSuccess: invalidateChanges,
  });
}

export function useRejectChangeMutation() {
  return useMutation({
    mutationFn: ({ id, driver }: { id: string; driver: string }) => api.changes(driver).reject(id),
    onSuccess: invalidateChanges,
  });
}

export function useScheduleChangeMutation() {
  return useMutation({
    mutationFn: ({
      id,
      driver,
      scheduledAt,
    }: {
      id: string;
      driver: string;
      scheduledAt: number;
    }) => api.changes(driver).schedule(id, scheduledAt),
    onSuccess: invalidateChanges,
  });
}
