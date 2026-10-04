import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { api, type Connection, type Table } from "@/lib/api";

interface AppState {
  // SQL connections drive the SQL pages (active one + its schema).
  connections: Connection[];
  active?: string;
  schema: Table[];
  busy: boolean;
  setActive: (name: string) => void;
  reload: () => Promise<void>;
  // Memory-cache connections (Redis and Valkey) are their own registry: the
  // engine families stay separate everywhere, including the shared state.
  memcacheConnections: Connection[];
  memcacheActive?: string;
  setMemcacheActive: (name: string) => void;
  reloadMemcache: () => Promise<void>;
}

const AppContext = createContext<AppState | null>(null);

// AppProvider owns the shared state (connections, the active one, its schema)
// so no page has to thread props or refetch.
export function AppProvider({ children }: { children: ReactNode }) {
  const [connections, setConnections] = useState<Connection[]>([]);
  const [active, setActive] = useState<string>();
  const [schema, setSchema] = useState<Table[]>([]);
  const [busy, setBusy] = useState(false);
  const [memcacheConnections, setMemcacheConnections] = useState<Connection[]>([]);
  const [memcacheActive, setMemcacheActive] = useState<string>();

  const reload = useCallback(async () => {
    setBusy(true);
    try {
      const { connections: list } = await api.connections.list();
      setConnections(list ?? []);
      setActive((prev) => (list?.some((c) => c.name === prev) ? prev : list?.[0]?.name));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, []);

  const reloadMemcache = useCallback(async () => {
    try {
      const { connections: list } = await api.memcache.list();
      setMemcacheConnections(list ?? []);
      setMemcacheActive((prev) => (list?.some((c) => c.name === prev) ? prev : list?.[0]?.name));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }, []);

  useEffect(() => {
    void reload();
    void reloadMemcache();
  }, [reload, reloadMemcache]);

  useEffect(() => {
    if (!active) {
      setSchema([]);
      return;
    }
    let cancelled = false;
    api.connections
      .schema(active)
      .then((res) => !cancelled && setSchema(res.tables ?? []))
      .catch(() => !cancelled && setSchema([]));
    return () => {
      cancelled = true;
    };
  }, [active]);

  const value = useMemo(
    () => ({
      connections,
      active,
      schema,
      busy,
      setActive,
      reload,
      memcacheConnections,
      memcacheActive,
      setMemcacheActive,
      reloadMemcache,
    }),
    [connections, active, schema, busy, reload, memcacheConnections, memcacheActive, reloadMemcache],
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp(): AppState {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp must be used inside AppProvider");
  return ctx;
}

export function useActiveConnection(): Connection | undefined {
  const { connections, active } = useApp();
  return connections.find((c) => c.name === active);
}
