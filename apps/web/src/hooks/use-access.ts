import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";

// Grants live under the engine path of the connection's driver, so every hook takes that driver rather than the console's capability grouping.

export function usePermissionsQuery(driver: string | undefined, name: string) {
  return useQuery({
    queryKey: queryKeys.access.permissions(driver, name),
    queryFn: () => api.permissions.list(driver as string, name),
    enabled: !!driver && !!name,
  });
}

export function useMembersQuery(enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.config.members,
    queryFn: () => api.config.members(),
    enabled,
  });
}

// useAccessMatrixQuery is the per-instance overview every admin page reads: one request, every connection, every grant.
export function useAccessMatrixQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.access.matrix,
    queryFn: () => api.permissions.matrix(),
    enabled,
  });
}

// useSetPermission grants a level and owns invalidation; the caller can add its own onSuccess (the editor clears the member input there).
export function useSetPermission(driver: string | undefined, name: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { member: string; memberType: string; level: string }) =>
      api.permissions.set(driver as string, name, input.member, input.memberType, input.level),
    onSuccess: (_data, input) => {
      toast.success(`${input.member} can ${input.level} ${name}`);
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.permissions(driver, name) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.matrix });
      void queryClient.invalidateQueries({ queryKey: queryKeys.me });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });
}

export function useRemovePermission(driver: string | undefined, name: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (member: string) => api.permissions.remove(driver as string, name, member),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.permissions(driver, name) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.matrix });
      void queryClient.invalidateQueries({ queryKey: queryKeys.me });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });
}
