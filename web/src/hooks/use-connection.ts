import { useEffect, useState } from "react";
import { useConnections } from "@/hooks/use-connections";
import { FAMILY } from "@/lib/families";

// useConnectionSelection gives a page its own instance choice, seeded from
// the shared active connection (the last one anyone picked) or an explicit
// seed. The list comes from the cached instances query.
export function useConnectionSelection(seed?: string) {
  const { connections, active, setActive, busy } = useConnections(FAMILY.SQL);
  const [connection, setConnection] = useState<string | undefined>(seed ?? active);

  useEffect(() => {
    if (!connection && connections.length > 0) setConnection(active ?? connections[0]?.name);
  }, [connection, connections, active]);

  function pick(name: string) {
    setConnection(name);
    setActive(name);
  }

  return { connection, connections, busy, pick };
}