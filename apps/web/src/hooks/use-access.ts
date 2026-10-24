import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";
import type { FamilyKey } from "@/lib/families";

export function usePermissionsQuery(family: FamilyKey | undefined, name: string) {
  return useQuery({
    queryKey: ["permissions", family, name],
    queryFn: () => api.permissions.list(family as FamilyKey, name),
    enabled: !!family && !!name,
  });
}

export function useMembersQuery(enabled: boolean) {
  return useQuery({
    queryKey: ["config", "members"],
    queryFn: () => api.config.members(),
    enabled,
  });
}

// useAccessMatrixQuery is the per-instance overview every admin page reads: one request, every connection, every grant.
export function useAccessMatrixQuery(enabled = true) {
  return useQuery({
    queryKey: ["access", "matrix"],
    queryFn: () => api.permissions.matrix(),
    enabled,
  });
}

// useSetPermission grants a level and owns invalidation; the caller can add its own onSuccess (the editor clears the member input there).
export function useSetPermission(family: FamilyKey | undefined, name: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: { member: string; memberType: string; level: string }) =>
      api.permissions.set(family as FamilyKey, name, input.member, input.memberType, input.level),
    onSuccess: (_data, input) => {
      toast.success(`${input.member} can ${input.level} ${name}`);
      void queryClient.invalidateQueries({ queryKey: ["permissions", family, name] });
      void queryClient.invalidateQueries({ queryKey: ["access", "matrix"] });
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });
}

export function useRemovePermission(family: FamilyKey | undefined, name: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (member: string) => api.permissions.remove(family as FamilyKey, name, member),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["permissions", family, name] });
      void queryClient.invalidateQueries({ queryKey: ["access", "matrix"] });
      void queryClient.invalidateQueries({ queryKey: ["me"] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });
}
