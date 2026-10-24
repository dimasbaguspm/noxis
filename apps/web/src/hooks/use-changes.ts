import { useMemo } from "react";
import { useDebounce } from "@/hooks/use-debounce";
import { useUrlState } from "@/hooks/use-url-state";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type ChangeInput } from "@/lib/api";
import { queryClient, queryKeys } from "@/lib/query-client";
import { classify } from "@/lib/statements";

export type ChangeKind = "dml" | "ddl";

// ./changes refetches the queue and any detail after an action, so a page never calls refetch() by hand.
function invalidateChanges() {
  void queryClient.invalidateQueries({ queryKey: ["sql", "changes"] });
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
  const { connection, status, page } = useChangeListUrlParams();
  const debounced = useDebounce(connection.trim());
  const connectionFilter = debounced || undefined;
  const statusFilter = status === "all" ? undefined : status;
  const query = useQuery({
    queryKey: queryKeys.sql.changes.list(kind, connectionFilter, statusFilter, page),
    queryFn: () => api.changes.list(connectionFilter, page, undefined, statusFilter),
  });
  const changes = useMemo(() => {
    if (!kind) return query.data?.changes;
    return query.data?.changes.filter((c) => (c.kind || classify(c.query ?? "")) === kind);
  }, [kind, query.data]);

  return { ...query, changes, total: query.data?.total };
}

export function useChangeGetQuery(id?: string) {
  return useQuery({
    queryKey: queryKeys.sql.changes.get(id ?? ""),
    queryFn: () => api.changes.get(id as string),
    enabled: !!id,
  });
}

export function useSubmitChangeMutation() {
  return useMutation({
    mutationFn: (input: ChangeInput) => api.changes.submit(input),
    onSuccess: invalidateChanges,
  });
}

export function useApproveChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.changes.approve(id),
    onSuccess: invalidateChanges,
  });
}

export function useApplyChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.changes.apply(id),
    onSuccess: invalidateChanges,
  });
}

export function useRejectChangeMutation() {
  return useMutation({
    mutationFn: (id: string) => api.changes.reject(id),
    onSuccess: invalidateChanges,
  });
}

export function useScheduleChangeMutation() {
  return useMutation({
    mutationFn: ({ id, scheduledAt }: { id: string; scheduledAt: number }) =>
      api.changes.schedule(id, scheduledAt),
    onSuccess: invalidateChanges,
  });
}

