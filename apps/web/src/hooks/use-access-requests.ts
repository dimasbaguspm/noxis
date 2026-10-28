import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { useUrlState } from "@/hooks/use-url-state";
import { queryKeys } from "@/lib/query-client";

const MY_ACCESS_DEFAULTS = { status: "all" };

// useMyAccessUrlParams is the status filter on /my/access, kept in the URL.
export function useMyAccessUrlParams() {
  const [state, patch] = useUrlState(MY_ACCESS_DEFAULTS);
  return { status: state.status, setStatus: (value: string) => patch({ status: value }) };
}

export function useMyRequestsQuery() {
  return useQuery({
    queryKey: queryKeys.access.requests.mine,
    queryFn: () => api.accessRequests.mine(),
  });
}

export function useToApproveQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.access.requests.toApprove,
    queryFn: () => api.accessRequests.toApprove(),
    enabled,
  });
}

// useDecideRequest approves or rejects a request and owns the invalidation, so the inbox stays a rendering surface.
export function useDecideRequest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, approve, note }: { id: string; approve: boolean; note?: string }) =>
      approve ? api.accessRequests.approve(id, note) : api.accessRequests.reject(id, note),
    onSuccess: (_data, vars) => {
      toast.success(vars.approve ? "request approved" : "request rejected");
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.requests.all });
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.all });
      void queryClient.invalidateQueries({ queryKey: queryKeys.me });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "decision failed"),
  });
}

export function useCreateRequest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { connection: string; level: string; reason?: string }) =>
      api.accessRequests.create(input.connection, input.level, input.reason ?? ""),
    onSuccess: (_data, vars) => {
      toast.success(`requested ${vars.level} on ${vars.connection}`);
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.requests.all });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "request failed"),
  });
}
