// Global console state (zustand): connections, the active one, and its schema.
import { create } from "zustand";
import { getSchema, listConnections, type Connection, type Table } from "./api/client";

interface State {
  connections: Connection[];
  active: string;
  schema: Table[];
  busy: boolean;
  pick: (name: string) => void;
  refresh: () => Promise<void>;
  setBusy: (b: boolean) => void;
}

export const useStore = create<State>((set, get) => ({
  connections: [],
  active: "",
  schema: [],
  busy: false,
  pick: (name) => {
    set({ active: name, schema: [] });
    void get().refresh();
  },
  setBusy: (b) => set({ busy: b }),
  refresh: async () => {
    const res = await listConnections();
    if (!res.ok) return;
    const connections = res.data?.connections ?? [];
    const stillThere = connections.some((c) => c.name === get().active);
    const active = stillThere ? get().active : (connections[0]?.name ?? "");
    set({ connections, active });
    if (!active) {
      set({ schema: [] });
      return;
    }
    const s = await getSchema(active);
    set({ schema: s.ok ? (s.data?.tables ?? []) : [] });
  },
}));
