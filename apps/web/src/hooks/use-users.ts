import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api";

export function useUsersQuery(q: string, status: string, enabled = true) {
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
