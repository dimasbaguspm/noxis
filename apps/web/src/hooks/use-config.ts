import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { queryKeys } from "@/lib/query-client";

// drivers is the server's hardcoded catalog, the only source for pickers.
export function useDriversQuery() {
  return useQuery({ queryKey: queryKeys.config.drivers, queryFn: () => api.config.drivers() });
}

// useLevelsQuery reads the backend's access-level catalog, so the console's pickers follow the server instead of keeping their own copy.
export function useLevelsQuery() {
  return useQuery({
    queryKey: queryKeys.config.levels,
    queryFn: () => api.config.levels(),
    staleTime: Infinity,
  });
}
