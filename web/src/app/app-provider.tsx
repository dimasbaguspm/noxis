import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { api, type Connection, type Table } from "@/lib/api";
import { FAMILIES, type FamilyKey } from "@/lib/families";

interface FamilyState {
  connections: Connection[];
  active?: string;
}

interface AppState {
  // registry is keyed by engine family: one store, not one per family.
  registry: Record<FamilyKey, FamilyState>;
  schema: Table[];
  busy: boolean;
  reload: (family: FamilyKey) => Promise<void>;
  setActive: (family: FamilyKey, name: string) => void;
}

const EMPTY: Record<FamilyKey, FamilyState> = {
  sql: { connections: [] },
  memcache: { connections: [] },
  mongodb: { connections: [] },
};

const AppContext = createContext<AppState | null>(null);

// AppProvider owns the connection registry, keyed by engine family, so no page
// threads props or refetches. A new family is another key, not another
// parallel useState trio.
export function AppProvider({ children }: { children: ReactNode }) {
  const [registry, setRegistry] = useState<Record<FamilyKey, FamilyState>>(EMPTY);
  const [schema, setSchema] = useState<Table[]>([]);
  const [busy, setBusy] = useState(false);

  const reload = useCallback(async (family: FamilyKey) => {
    setBusy(true);
    try {
      const { connections: list } = await api.family(family).list();
      setRegistry((prev) => {
        const current = prev[family];
        const active = list?.some((c) => c.name === current.active) ? current.active : list?.[0]?.name;
        return { ...prev, [family]: { connections: list ?? [], active } };
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, []);

  const setActive = useCallback((family: FamilyKey, name: string) => {
    setRegistry((prev) => ({ ...prev, [family]: { ...prev[family], active: name } }));
  }, []);

  useEffect(() => {
    for (const family of FAMILIES) void reload(family.key);
  }, [reload]);

  const activeSQL = registry.sql.active;
  useEffect(() => {
    if (!activeSQL) {
      setSchema([]);
      return;
    }
    let cancelled = false;
    api.connections
      .schema(activeSQL)
      .then((res) => !cancelled && setSchema(res.tables ?? []))
      .catch(() => !cancelled && setSchema([]));
    return () => {
      cancelled = true;
    };
  }, [activeSQL]);

  const value = useMemo(
    () => ({ registry, schema, busy, reload, setActive }),
    [registry, schema, busy, reload, setActive],
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp(): AppState {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp must be used inside AppProvider");
  return ctx;
}

// useFamily gives a consumer its family's connections with the family already
// bound, so the key never has to travel further.
export function useFamily(family: FamilyKey) {
  const { registry, busy, reload, setActive } = useApp();
  const state = registry[family];
  return {
    connections: state.connections,
    active: state.active,
    busy,
    setActive: useCallback((name: string) => setActive(family, name), [family, setActive]),
    reload: useCallback(() => reload(family), [family, reload]),
  };
}
