import { useEffect, useState } from "react";
import { api, type Table } from "@/lib/api";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";

// useConnectionSelection gives a page its own instance choice, seeded from
// the provider's default (the last one anyone picked).
export function useConnectionSelection() {
  const { connections, active, setActive } = useApp();
  const [connection, setConnection] = useState<string | undefined>(active);

  useEffect(() => {
    if (!connection && connections.length > 0) setConnection(active ?? connections[0]?.name);
  }, [connection, connections, active]);

  function pick(name: string) {
    setConnection(name);
    setActive(name);
  }

  return { connection, pick };
}

// useSchema loads the tables of one connection for a page.
export function useSchema(connection?: string) {
  const [tables, setTables] = useState<Table[]>([]);
  const load = useAction(async (name: string) => {
    const res = await api.connections.schema(name);
    setTables(res.tables ?? []);
  });

  useEffect(() => {
    if (!connection) {
      setTables([]);
      return;
    }
    void load.run(connection);
  }, [connection, load.run]);

  return { tables, loading: load.loading };
}
