import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";

export function useMyRequestsQuery() {
  return useQuery({
    queryKey: ["access-requests", "mine"],
    queryFn: () => api.accessRequests.mine(),
  });
}

export function useToApproveQuery(enabled = true) {
  return useQuery({
    queryKey: ["access-requests", "to-approve"],
    queryFn: () => api.accessRequests.toApprove(),
    enabled,
  });
}

// useDecideRequest approves or rejects a request and owns the invalidation, so
// the inbox stays a rendering surface.
export function useDecideRequest() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, approve }: { id: string; approve: boolean }) =>
      approve ? api.accessRequests.approve(id) : api.accessRequests.reject(id),
    onSuccess: (_data, vars) => {
      toast.success(vars.approve ? "request approved" : "request rejected");
      void queryClient.invalidateQueries({ queryKey: ["access-requests"] });
      void queryClient.invalidateQueries({ queryKey: ["permissions"] });
      void queryClient.invalidateQueries({ queryKey: ["me"] });
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
      void queryClient.invalidateQueries({ queryKey: ["access-requests"] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "request failed"),
  });
}
