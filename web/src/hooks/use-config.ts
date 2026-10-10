import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";

// drivers is the server's hardcoded catalog, the only source for pickers.
export function useDriversQuery() {
  return useQuery({ queryKey: queryKeys.config.drivers, queryFn: () => api.config.drivers() });
}