import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";

// useInstanceCountsQuery reads how many instances each driver serves, in one call, for the sidebar's driver list.
export function useInstanceCountsQuery() {
  return useQuery({
    queryKey: queryKeys.instances.counts,
    queryFn: () => api.instances.counts(),
  });
}

// countLabel caps a driver's count at 99+, so a busy engine cannot widen the sidebar row.
export function countLabel(count: number | undefined): string {
  if (!count) return "";
  return count > 99 ? "99+" : String(count);
}
