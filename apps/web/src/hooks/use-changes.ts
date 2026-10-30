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

const CHANGE_DEFAULTS = { q: "", status: "all", page: "1" };

// useChangeListUrlParams is the change queue's filter state, kept in the URL.
export function useChangeListUrlParams() {
  const [state, patch] = useUrlState(CHANGE_DEFAULTS);
  return {
    connection: state.q,
    status: state.status,
    page: Number(state.page),
    setConnection: (value: string) => patch({ q: value, page: "1" }),
    setStatus: (value: string) => patch({ status: value, page: "1" }),
    setPage: (value: number) => patch({ page: String(value) }),
  };
}

// useChangeListQuery filters the shared server list by URL state so DML and DDL tabs stay distinct.
export function useChangeListQuery(kind?: ChangeKind) {
  const { driver = "" } = useParams();
  const { connection, status, page } = useChangeListUrlParams();
  const debounced = useDebounce(connection.trim());
  const connectionFilter = debounced || undefined;
  const statusFilter = status === "all" ? undefined : status;
  const query = useQuery({
    queryKey: queryKeys.changes.list(driver, kind, connectionFilter, statusFilter, page),
    // the server filters by kind before it paginates, so the total is the queue's own count
    queryFn: () => api.changes(driver).list(connectionFilter, page, undefined, statusFilter, kind),
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
