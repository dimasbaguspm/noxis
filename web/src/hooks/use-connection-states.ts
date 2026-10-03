import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export interface ConnectionState {
  name: string;
  ok: boolean;
  latencyMs: number;
  error?: string;
}

// useConnectionStates streams reachability for every registered connection —
// the dashboard shows it per visible row (SSE, one tick per 5s).
export function useConnectionStates() {
  const [states, setStates] = useState<Record<string, ConnectionState>>({});
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const source = new EventSource("/v1/sql/connections/status/stream");
    source.addEventListener("open", () => setConnected(true));
    source.addEventListener("error", () => setConnected(false));
    source.addEventListener("status", (event) => {
      try {
        const list = JSON.parse((event as MessageEvent).data) as ConnectionState[];
        const next: Record<string, ConnectionState> = {};
        for (const state of list) next[state.name] = state;
        setStates(next);
      } catch {
        /* ignore malformed frames */
      }
    });
    return () => {
      source.close();
      setConnected(false);
    };
  }, []);

  return { states, connected };
}

// api is imported for the stream path so the endpoint lives in one place.
export const statusStreamPath = api.connections.statusStream();
