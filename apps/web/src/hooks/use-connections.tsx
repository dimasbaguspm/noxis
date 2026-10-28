import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import { FAMILIES, type FamilyKey } from "@/lib/families";
import { queryKeys } from "@/lib/query-client";

// ActiveProvider remembers the per-family connection choice in memory only; it holds no data.

const DEFAULT_ACTIVE = FAMILIES.reduce(
  (acc, f) => ({ ...acc, [f.key]: undefined }),
  {} as Partial<Record<FamilyKey, string>>,
);

interface ActiveState {
  active: Partial<Record<FamilyKey, string>>;
  setActive: (family: FamilyKey, name: string) => void;
}

const ActiveContext = createContext<ActiveState | null>(null);

export function ActiveProvider({ children }: { children: ReactNode }) {
  const [active, set] = useState<Partial<Record<FamilyKey, string>>>(DEFAULT_ACTIVE);
  const setActive = useCallback((family: FamilyKey, name: string) => {
    set((prev) => ({ ...prev, [family]: name }));
  }, []);
  const value = useMemo(() => ({ active, setActive }), [active, setActive]);
  return <ActiveContext.Provider value={value}>{children}</ActiveContext.Provider>;
}

export function useSetActiveConnection() {
  const ctx = useContext(ActiveContext);
  if (!ctx) throw new Error("useSetActiveConnection must be used inside ActiveProvider");
  return ctx.setActive;
}

// useConnections gives a family its connections with the shared active choice bound.
export function useConnections(family: FamilyKey) {
  const ctx = useContext(ActiveContext);
  if (!ctx) throw new Error("useConnections must be used inside ActiveProvider");
  const queryClient = useQueryClient();
  const list = usePagedInstancesQuery();
  const connections = (list.data?.instances ?? []).filter((c) => c.family === family);
  const active = ctx.active[family] ?? connections[0]?.name;
  const setActive = useCallback((name: string) => ctx.setActive(family, name), [ctx, family]);
  const reload = useCallback(
    () => void queryClient.invalidateQueries({ queryKey: queryKeys.instances.all }),
    [queryClient],
  );
  return { connections, active, setActive, reload, busy: list.isPending };
}
