import { useMutation, useQuery } from "@tanstack/react-query";
import { api, type InstanceQuery } from "@/lib/api";
import { queryClient, queryKeys } from "@/lib/query-client";

// usePagedInstancesQuery is the cross-family instance list (dashboard + the
// global registry reads), paged with the same params the API exposes.
export function usePagedInstancesQuery(query: InstanceQuery = {}) {
  return useQuery({
    queryKey: queryKeys.instances.list(query),
    queryFn: () => api.instances.list(query),
  });
}

export function useInstanceQuery(id?: string) {
  return useQuery({
    queryKey: queryKeys.instances.get(id ?? ""),
    queryFn: () => api.instances.get(id as string),
    enabled: !!id,
  });
}

export function useRotatePasswordMutation() {
  return useMutation({
    mutationFn: ({ id, password }: { id: string; password: string }) =>
      api.instances.rotatePassword(id, password),
    onSuccess: (_data, { id }) => {
      void queryClient.invalidateQueries({ queryKey: ["instances"] });
      void queryClient.invalidateQueries({ queryKey: queryKeys.instances.get(id) });
      // the console's own permission view follows the new permissions
      void queryClient.invalidateQueries({ queryKey: queryKeys.me });
    },
  });
}