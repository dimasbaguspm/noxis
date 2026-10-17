import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { useUrlState } from "@/hooks/use-url-state";

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
    queryKey: ["users", q, status],
    queryFn: () => api.users.list(q, status),
    enabled,
  });
}

export function useUserQuery(subject: string) {
  return useQuery({
    queryKey: ["user", subject],
    queryFn: () => api.users.get(subject),
    enabled: subject !== "",
  });
}

// useSetUserDisabled toggles an account and owns the cache invalidation, so the
// page stays a rendering surface.
export function useSetUserDisabled(subject: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (disabled: boolean) => api.users.setDisabled(subject, disabled),
    onSuccess: () => {
      toast.success("account updated");
      void queryClient.invalidateQueries({ queryKey: ["user", subject] });
      void queryClient.invalidateQueries({ queryKey: ["users"] });
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : "update failed"),
  });
}
