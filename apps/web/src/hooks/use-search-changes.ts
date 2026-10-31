import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryClient, queryKeys } from "@/lib/query-client";

// useSearchChangesQuery lists the search changes of one driver. The queue is its own, so it asks for the kind it owns
// rather than sharing a page shaped for documents.
export function useSearchChangesQuery(driver?: string, connection?: string, status?: string) {
  const query = useQuery({
    queryKey: queryKeys.search.requests(driver, connection, status),
    queryFn: () => api.search(driver as string).changes.list(connection, status),
    enabled: !!driver,
  });
  return {
    ...query,
    changes: query.data?.changes ?? [],
    total: query.data?.total ?? 0,
  };
}

// useSearchChangeQuery opens one change: the query, the impact the platform reported, and its checks.
export function useSearchChangeQuery(driver?: string, id?: string) {
  return useQuery({
    queryKey: queryKeys.search.request(driver, id),
    queryFn: () => api.search(driver as string).changes.get(id as string),
    enabled: !!driver && !!id,
  });
}

// useSearchChangeAction decides one change, then refreshes the queue and the change itself so the page never shows a
// status the server has already moved past.
export function useSearchChangeAction(driver?: string) {
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: "approve" | "apply" | "reject" }) =>
      api.search(driver as string).changes[action](id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.search.requests(driver) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.search.request(driver) });
    },
  });
}
