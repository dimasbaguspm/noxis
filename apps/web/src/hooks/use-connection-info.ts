import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

// The version is detected from the server and never typed, so it cannot be invalidated by a write the way a schema can; it only goes stale when the server changes under us.
const infoKey = (name: string) => ["console", "instances", "info", name] as const;

// useConnectionInfo reads what a connection's server reports about itself, for the badge on the instance page. The driver is part of the path, so nothing is read until both are known.
export function useConnectionInfo(driver: string | undefined, name: string | undefined) {
  return useQuery({
    queryKey: infoKey(name ?? ""),
    queryFn: () => api.connections(driver as string).info(name as string),
    enabled: Boolean(driver) && Boolean(name),
    staleTime: 5 * 60_000,
  });
}
