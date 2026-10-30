import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useDriversQuery } from "@/hooks/use-config";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import { capabilitiesOf, type Capability } from "@/lib/console-nav";
import { queryKeys } from "@/lib/query-client";

// ActiveProvider remembers the per-capability connection choice in memory only; it holds no data.
const DEFAULT_ACTIVE: Partial<Record<Capability, string>> = {};

interface ActiveState {
  active: Partial<Record<Capability, string>>;
  setActive: (capability: Capability, name: string) => void;
}

const ActiveContext = createContext<ActiveState | null>(null);

export function ActiveProvider({ children }: { children: ReactNode }) {
  const [active, set] = useState<Partial<Record<Capability, string>>>(DEFAULT_ACTIVE);
  const setActive = useCallback((capability: Capability, name: string) => {
    set((prev) => ({ ...prev, [capability]: name }));
  }, []);
  const value = useMemo(() => ({ active, setActive }), [active, setActive]);
  return <ActiveContext.Provider value={value}>{children}</ActiveContext.Provider>;
}

export function useSetActiveConnection() {
  const ctx = useContext(ActiveContext);
  if (!ctx) throw new Error("useSetActiveConnection must be used inside ActiveProvider");
  return ctx.setActive;
}

// useConnections gives a capability its connections with the shared active choice bound. Whether a driver has the
// capability is the catalog's call, so a driver the console has never seen still lands in the right list; every engine
// call takes the driver of the connection it acts on, so the active driver comes along.
export function useConnections(capability: Capability) {
  const ctx = useContext(ActiveContext);
  if (!ctx) throw new Error("useConnections must be used inside ActiveProvider");
  const queryClient = useQueryClient();
  const list = usePagedInstancesQuery();
  const drivers = useDriversQuery();
  const connections = (list.data?.instances ?? []).filter((c) =>
    capabilitiesOf(drivers.data, c.driver).includes(capability),
  );
  const active = ctx.active[capability] ?? connections[0]?.name;
  const driver = connections.find((c) => c.name === active)?.driver;
  const setActive = useCallback(
    (name: string) => ctx.setActive(capability, name),
    [ctx, capability],
  );
  const reload = useCallback(
    () => void queryClient.invalidateQueries({ queryKey: queryKeys.instances.all }),
    [queryClient],
  );
  return {
    connections,
    active,
    driver,
    setActive,
    reload,
    busy: list.isPending || drivers.isPending,
  };
}
