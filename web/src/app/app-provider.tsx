import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { api, type Connection, type DriverInfo, type Table } from "@/lib/api";
import { useAuth } from "@/app/auth-provider";
import { FAMILIES, type FamilyKey } from "@/lib/families";

interface FamilyState {
  connections: Connection[];
  active?: string;
}

interface AppState {
  // registry is keyed by engine family: one store, not one per family.
  registry: Record<FamilyKey, FamilyState>;
  // drivers is the server's hardcoded catalog, the only source for pickers.
  drivers: DriverInfo[];
  schema: Table[];
  busy: boolean;
  reload: (family: FamilyKey) => Promise<void>;
  setActive: (family: FamilyKey, name: string) => void;
}

const EMPTY = FAMILIES.reduce(
  (acc, family) => ({ ...acc, [family.key]: { connections: [] } }),
  {} as Record<FamilyKey, FamilyState>,
);

const AppContext = createContext<AppState | null>(null);

// AppProvider owns the connection registry, keyed by engine family, so no page
// threads props or refetches. A new family is another key, not another
// parallel useState trio. It also bootstraps OIDC and the caller's levels.
export function AppProvider({ children }: { children: ReactNode }) {
  const [registry, setRegistry] = useState<Record<FamilyKey, FamilyState>>(EMPTY);
  const [drivers, setDrivers] = useState<DriverInfo[]>([]);
  const [schema, setSchema] = useState<Table[]>([]);
  const [pending, setPending] = useState(0);
  const { ready } = useAuth();

  // reload fetches every family in one call and splits the result, so the
  // console no longer issues one request per family. The argument is kept so
  // family-scoped callers (the connection dialog) stay unchanged.
  const reload = useCallback(async (_family?: FamilyKey) => {
    setPending((n) => n + 1);
    try {
      const { instances } = await api.instances.list();
      setRegistry((prev) => {
        const next = { ...prev };
        for (const family of FAMILIES) {
          const list = (instances ?? []).filter((c) => c.family === family.key);
          const current = prev[family.key];
          const active = list.some((c) => c.name === current.active) ? current.active : list[0]?.name;
          next[family.key] = { connections: list, active };
        }
        return next;
      });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setPending((n) => n - 1);
    }
  }, []);

  const setActive = useCallback((family: FamilyKey, name: string) => {
    setRegistry((prev) => ({ ...prev, [family]: { ...prev[family], active: name } }));
  }, []);

  useEffect(() => {
    if (!ready) return;
    void reload();
    api.config
      .drivers()
      .then(setDrivers)
      .catch(() => setDrivers([]));
  }, [ready, reload]);

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
    () => ({
      registry,
      drivers,
      schema,
      busy: pending > 0,
      reload,
      setActive,
    }),
    [registry, drivers, schema, pending, reload, setActive],
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
