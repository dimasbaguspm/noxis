import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, type UserRole, type UserStatus } from "@/lib/api";
import { useUrlState } from "@/hooks/use-url-state";
import { queryKeys } from "@/lib/query-client";

const USERS_DEFAULTS = { q: "", status: "all" };

// useUsersUrlParams is the directory's search and status, kept in the URL.
export function useUsersUrlParams() {
  const [state, patch] = useUrlState(USERS_DEFAULTS);
  return {
    q: state.q,
    status: state.status,
    setQ: (value: string) => patch({ q: value }),
    setStatus: (value: string) => patch({ status: value }),
  };
}

// useUsersQuery reads the directory for the URL filters.
export function useUsersQuery(enabled = true) {
  const { q, status } = useUsersUrlParams();
  return useQuery({
    queryKey: queryKeys.users.list(q, status),
    queryFn: () => api.users.list(q, status),
    enabled,
  });
}

export function useUserQuery(subject: string) {
  return useQuery({
    queryKey: queryKeys.users.get(subject),
    queryFn: () => api.users.get(subject),
    enabled: subject !== "",
  });
}

// useUpdateUser owns the status/role mutation and its cache invalidation, so the pages stay rendering surfaces.
export function useUpdateUser(subject: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (patch: { status?: UserStatus; role?: UserRole }) =>
      api.users.update(subject, patch),
    onSuccess: (_data, patch) => {
      toast.success(
        patch.role ? `role changed to ${patch.role}` : `account is now ${patch.status}`,
      );
      void queryClient.invalidateQueries({ queryKey: queryKeys.users.get(subject) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.users.all });
      void queryClient.invalidateQueries({ queryKey: queryKeys.access.matrix });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "update failed"),
  });
}
