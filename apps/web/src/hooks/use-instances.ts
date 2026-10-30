import { useMutation, useQuery } from "@tanstack/react-query";
import { useUrlState } from "@/hooks/use-url-state";
import { api, type InstanceQuery } from "@/lib/api";
import { queryClient, queryKeys } from "@/lib/query-client";

export const ALL_DRIVERS = "all";
const INSTANCE_DEFAULTS = { q: "", driver: ALL_DRIVERS, page: "1" };

// useInstancesUrlParams is the dashboard's filter state, kept in the URL: search, driver and page.
export function useInstancesUrlParams() {
  const [state, patch] = useUrlState(INSTANCE_DEFAULTS);
  return {
    q: state.q,
    driver: state.driver,
    page: Number(state.page),
    setQ: (value: string) => patch({ q: value, page: "1" }),
    setDriver: (value: string) => patch({ driver: value, page: "1" }),
    setPage: (value: number) => patch({ page: String(value) }),
  };
}

// usePagedInstancesQuery is the cross-driver instance list (dashboard + the global registry reads), paged with the same params the API exposes.
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
      void queryClient.invalidateQueries({ queryKey: queryKeys.instances.all });
      void queryClient.invalidateQueries({ queryKey: queryKeys.instances.get(id) });
      // the console's own permission view follows the new permissions
      void queryClient.invalidateQueries({ queryKey: queryKeys.me });
    },
  });
}
