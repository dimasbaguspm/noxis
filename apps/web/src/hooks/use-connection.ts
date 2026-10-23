import { useState } from "react";
import { useConnections } from "@/hooks/use-connections";
import { FAMILY } from "@/lib/families";

// useConnectionSelection gives a page its own instance choice, seeded from
// the shared active connection (the last one anyone picked) or an explicit
// seed. The list comes from the cached instances query.
export function useConnectionSelection(seed?: string) {
  const { connections, active, setActive, busy } = useConnections(FAMILY.SQL);
  const [selected, setSelected] = useState<string | undefined>(seed);
  const connection = selected ?? active ?? connections[0]?.name;

  function pick(name: string) {
    setSelected(name);
    setActive(name);
  }

  return { connection, connections, busy, pick };
}