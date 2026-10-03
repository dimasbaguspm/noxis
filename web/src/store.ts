// Global console state (zustand): the active node, the live node list, and
// the assigned engines. Pages subscribe; refresh() is the 5s heartbeat poll.
import { create } from "zustand";
import { getSpec, getStatus, type Engine, type NodeStatus } from "./api/client";

interface ConsoleState {
  nodes: NodeStatus[];
  node: string;
  engines: Engine[];
  detail: string;
  busy: boolean;
  pick: (node: string) => void;
  refresh: () => Promise<void>;
  setBusy: (b: boolean) => void;
}

export const useConsole = create<ConsoleState>((set, get) => ({
  nodes: [],
  node: "",
  engines: [],
  detail: "",
  busy: false,
  pick: (node) => {
    set({ node, detail: "" });
    void get().refresh();
  },
  setBusy: (b) => set({ busy: b }),
  refresh: async () => {
    const res = await getStatus();
    if (!res.ok) return;
    const nodes = res.data?.nodes ?? [];
    const node = get().node || nodes[0]?.node || "";
    set({ nodes, node });
    if (!node) return;
    const s = await getSpec(node);
    const engines = s.ok ? (s.data?.engines ?? []) : [];
    set({
      engines,
      detail: engines.length ? `${engines.length} engine(s) assigned` : "no spec assigned",
    });
  },
}));