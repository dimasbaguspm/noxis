import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

// The version is detected from the server and never typed, so it cannot be invalidated by a write the way a schema can; it only goes stale when the server changes under us.
const infoKey = (name: string) => ["console", "instances", "info", name] as const;

// useConnectionInfo reads what a connection's server reports about itself, for the badge on the instance page.
export function useConnectionInfo(name: string | undefined) {
  return useQuery({
    queryKey: infoKey(name ?? ""),
    queryFn: () => api.connections.info(name as string),
    enabled: Boolean(name),
    staleTime: 5 * 60_000,
  });
}
