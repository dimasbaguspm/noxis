import { useMemo } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type ChangeInput } from "@/lib/api";
import { queryClient, queryKeys } from "@/lib/query-client";
import { classify } from "@/lib/statements";

export type ChangeKind = "dml" | "ddl";

// ./changes refetches the queue and any detail after an action, so a page
// never calls refetch() by hand.
function invalidateChanges() {
  void queryClient.invalidateQueries({ queryKey: ["sql", "changes"] });
}

// useChangeListQuery lists a page of changes (the server returns one list for
// dml and ddl; the kind filter is applied here, mirroring the pre-query
// behavior so no change disappears from either tab).
export function useChangeListQuery(
  kind?: ChangeKind,
  connection?: string,
  status?: string,
  page = 1,
) {
  const query = useQuery({
    queryKey: queryKeys.sql.changes.list(kind, connection, status, page),
    queryFn: () => api.changes.list(connection, page, undefined, status),
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

