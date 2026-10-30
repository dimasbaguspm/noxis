import { useCallback } from "react";
import { useParams, useSearchParams } from "react-router-dom";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import type { Connection } from "@/lib/api";

export interface DriverConnections {
  driver: string;
  connections: Connection[];
  connection?: string;
  pick: (name: string) => void;
  busy: boolean;
}

// useDriverConnections is a page's instance list and choice: only the connections of the driver in the URL, and the
// instance the URL names, falling back to the first one. The instance lives in ?instance=, so a reload or a copied link
// reproduces the page and a page can never reach another driver's instance.
export function useDriverConnections(): DriverConnections {
  const { driver = "" } = useParams();
  const [params, setParams] = useSearchParams();
  const list = usePagedInstancesQuery({ driver: driver || undefined, perPage: 100 });
  const connections = list.data?.instances ?? [];
  const named = params.get("instance") ?? "";
  const connection = connections.find((c) => c.name === named)?.name ?? connections[0]?.name;
  const pick = useCallback(
    (name: string) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev);
          if (name) next.set("instance", name);
          else next.delete("instance");
          return next;
        },
        { replace: true },
      );
    },
    [setParams],
  );
  return { driver, connections, connection, pick, busy: list.isPending };
}
