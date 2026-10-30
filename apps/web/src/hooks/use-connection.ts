import { useState } from "react";
import { useConnections } from "@/hooks/use-connections";
import { CAPABILITY } from "@/lib/console-nav";

// useConnectionSelection gives a page its own instance choice, seeded from the shared active connection (the last one anyone picked) or an explicit seed. The driver travels with the choice: the API addresses an engine by driver, not by the console's grouping of it.
export function useConnectionSelection(seed?: string, idSeed?: string) {
  const { connections, active, setActive, busy } = useConnections(CAPABILITY.SQL);
  const [selected, setSelected] = useState<string | undefined>(seed);
  const seeded = connections.find((c) => c.id === idSeed)?.name ?? seed;
  const connection = selected ?? seeded ?? active ?? connections[0]?.name;
  // the driver belongs to the resolved connection, not to the capability's last pick: a seeded or linked instance may differ
  const driver = connections.find((c) => c.name === connection)?.driver;

  function pick(name: string) {
    setSelected(name);
    setActive(name);
  }

  return { connection, driver, connections, busy, pick };
}
