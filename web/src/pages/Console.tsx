import { useEffect, useState } from "react";
import * as Select from "@radix-ui/react-select";
import { ENGINE_TYPES, putSpec, runOp, type Engine, type OpResult } from "../api/client";
import { useConsole } from "../store";

const OPS = ["query", "alter", "rollback", "analytics"] as const;
const COLORS: Record<string, string> = {
  up: "bg-[#3fb950]",
  online: "bg-[#3fb950]",
  error: "bg-[#f85149]",
  offline: "bg-[#8b949e]",
};

export default function Console() {
  const { nodes, node, engines, detail, busy, pick, refresh, setBusy } = useConsole();
  const [op, setOp] = useState<(typeof OPS)[number]>("query");
  const [engine, setEngine] = useState("");
  const [sql, setSql] = useState("");
  const [out, setOut] = useState<OpResult | null>(null);
  const [outErr, setOutErr] = useState(false);
  const [name, setName] = useState("");
  const [type, setType] = useState("postgres");
  const [version, setVersion] = useState("");
  const [ports, setPorts] = useState("");

  useEffect(() => {
    void refresh();
    const t = setInterval(() => void refresh(), 5000);
    return () => clearInterval(t);
  }, [refresh]);

  useEffect(() => {
    setVersion(ENGINE_TYPES[type]?.[0] ?? "");
  }, [type]);

  useEffect(() => {
    setEngine((e) => (engines.some((x) => x.name === e) ? e : (engines[0]?.name ?? "")));
  }, [engines]);

  async function save(next: Engine[]) {
    if (!node) return;
    setBusy(true);
    const res = await putSpec(node, next);
    setBusy(false);
    if (!res.ok) {
      setOutErr(true);
      setOut({ error: "save failed: " + String((res.data as unknown as { error?: string })?.error ?? "") });
      return;
    }
    await refresh();
  }

  async function add() {
    if (!name.trim() || !node) return;
    const e: Engine = { name: name.trim(), type, version: version.trim() };
    if (ports.trim()) {
      e.ports = {};
      for (const p of ports.split(",")) {
        const [h, c] = p.trim().split(":");
        if (h && c) e.ports[h] = c;
      }
    }
    await save([...engines, e]);
    setName("");
    setPorts("");
  }

  async function run() {
    if (!node || !engine || !sql.trim()) return;
    setBusy(true);
    const body: Record<string, string> =
      op === "analytics" ? { engine, table: sql.trim() } : { engine, sql: sql.trim() };
    const res = await runOp(node, op, body);
    setBusy(false);
    setOutErr(!res.ok);
    setOut(res.data ?? { error: `HTTP ${res.status}` });
  }

  const current = nodes.find((n) => n.node === node);

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9]">
      <header className="flex items-center gap-3 border-b border-[#30363d] bg-[#161b22] px-4 py-2.5">
        <h1 className="m-0 text-[15px] font-semibold text-[#58a6ff]">noxis console</h1>
        <div className="flex flex-wrap gap-1.5">
          {nodes.map((n) => (
            <button
              key={n.node}
              onClick={() => n.node && pick(n.node)}
              className={`rounded-full border px-3 py-1 text-sm ${
                n.node === node
                  ? "border-[#58a6ff] text-[#58a6ff]"
                  : "border-[#30363d] bg-[#0d1117] hover:border-[#58a6ff]"
              }`}
            >
              <span className={`mr-1.5 inline-block h-2 w-2 rounded-full ${COLORS[n.state ?? ""] ?? "bg-[#8b949e]"}`} />
              {n.node}
            </button>
          ))}
        </div>
      </header>

      <main className="mx-auto grid max-w-[1400px] gap-4 p-4 lg:grid-cols-[380px_1fr]">
        {/* engines */}
        <section className="rounded-lg border border-[#30363d] bg-[#161b22] p-4">
          <h2 className="mb-2.5 text-[13px] uppercase tracking-wider text-[#8b949e]">engines — {node}</h2>
          <table className="w-full border-collapse text-[13px]">
            <thead>
              <tr className="text-left text-[#8b949e]">
                <th className="border-b border-[#21262d] py-1.5 pr-2">name</th>
                <th className="border-b border-[#21262d] py-1.5 pr-2">type</th>
                <th className="border-b border-[#21262d] py-1.5 pr-2">version</th>
                <th className="border-b border-[#21262d] py-1.5 pr-2">ports</th>
                <th className="border-b border-[#21262d] py-1.5" />
              </tr>
            </thead>
            <tbody>
              {engines.map((e, i) => (
                <tr key={i}>
                  <td className="border-b border-[#21262d] py-1.5 pr-2">{e.name}</td>
                  <td className="border-b border-[#21262d] py-1.5 pr-2">{e.type}</td>
                  <td className="border-b border-[#21262d] py-1.5 pr-2">{e.version}</td>
                  <td className="border-b border-[#21262d] py-1.5 pr-2">
                    {Object.entries(e.ports ?? {}).map(([h, c]) => `${h}:${c}`).join(", ")}
                  </td>
                  <td className="border-b border-[#21262d] py-1.5 text-right">
                    <button onClick={() => void save(engines.filter((_, j) => j !== i))} className="text-[#f85149]">
                      ✕
                    </button>
                  </td>
                </tr>
              ))}
              {engines.length === 0 && (
                <tr><td colSpan={5} className="py-2 text-xs text-[#8b949e]">no engines assigned</td></tr>
              )}
            </tbody>
          </table>

          <div className="mt-2.5 flex gap-2">
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="name (db1)"
              className="w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <select value={type} onChange={(e) => setType(e.target.value)}
              className="w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]">
              {Object.keys(ENGINE_TYPES).map((k) => <option key={k}>{k}</option>)}
            </select>
            <input value={version} onChange={(e) => setVersion(e.target.value)} placeholder="version"
              className="w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
          </div>
          <div className="mt-2 flex gap-2">
            <input value={ports} onChange={(e) => setPorts(e.target.value)} placeholder="ports host:container, comma sep"
              className="w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <button onClick={() => void add()} disabled={busy}
              className="rounded-md bg-[#1f6feb] px-4 py-1.5 text-white disabled:opacity-50">add</button>
          </div>
          <div className="mt-2 flex gap-2">
            <button onClick={() => void save(engines)} disabled={busy}
              className="rounded-md border border-[#30363d] bg-[#21262d] px-4 py-1.5 disabled:opacity-50">save spec</button>
            <button onClick={() => void save([])} disabled={busy}
              className="rounded-md border border-[#30363d] bg-[#21262d] px-4 py-1.5 disabled:opacity-50">stop engines</button>
          </div>
          <p className="mt-2 text-xs text-[#8b949e]">{detail}</p>
        </section>

        {/* doorman */}
        <section className="rounded-lg border border-[#30363d] bg-[#161b22] p-4">
          <h2 className="mb-2.5 text-[13px] uppercase tracking-wider text-[#8b949e]">doorman — {node}</h2>
          <div className="mb-2 flex gap-1.5">
            {OPS.map((o) => (
              <button key={o} onClick={() => setOp(o)}
                className={`rounded-t-md border-b-0 border border-[#30363d] px-3 py-1 text-sm ${
                  op === o ? "border-b-transparent bg-[#1f6feb] text-white" : "bg-[#21262d]"
                }`}>{o}</button>
            ))}
          </div>

          <Select.Root value={engine} onValueChange={setEngine}>
            <Select.Trigger className="mb-2 flex w-full items-center justify-between rounded-md border border-[#30363d] bg-[#0d1117] px-3 py-2 outline-none focus:border-[#58a6ff]">
              <Select.Value placeholder="(no engine)" />
              <Select.Icon>▾</Select.Icon>
            </Select.Trigger>
            <Select.Portal>
              <Select.Content className="rounded-md border border-[#30363d] bg-[#161b22] text-[#c9d1d9]">
                <Select.Viewport>
                  {engines.map((e, i) => (
                    <Select.Item key={e.name ?? String(i)} value={e.name ?? ""}
                      className="cursor-pointer px-3 py-1.5 outline-none data-[highlighted]:bg-[#21262d]">
                      <Select.ItemText>{e.name}</Select.ItemText>
                    </Select.Item>
                  ))}
                </Select.Viewport>
              </Select.Content>
            </Select.Portal>
          </Select.Root>

          <textarea value={sql} onChange={(e) => setSql(e.target.value)}
            placeholder={op === "analytics" ? "table name" : "SQL"}
            onKeyDown={(e) => { if ((e.metaKey || e.ctrlKey) && e.key === "Enter") void run(); }}
            className="min-h-[140px] w-full resize-y rounded-md border border-[#30363d] bg-[#0d1117] px-3 py-2 outline-none focus:border-[#58a6ff]" />
          <button onClick={() => void run()} disabled={busy}
            className="mt-2 rounded-md bg-[#1f6feb] px-4 py-1.5 text-white disabled:opacity-50">run</button>
          <pre className={`mt-2 min-h-[60px] whitespace-pre-wrap break-words rounded-md border border-[#30363d] bg-[#0d1117] p-2.5 ${outErr ? "text-[#f85149]" : "text-[#3fb950]"}`}>
            {out ? (out.error ?? out.output ?? "") : ""}
          </pre>
          <p className="mt-2 text-xs text-[#8b949e]">
            {current ? `${current.state} @ ${new Date((current.ts ?? 0) * 1000).toLocaleString()}` : "no nodes registered"}
          </p>
          <p className="text-xs text-[#8b949e]">{current?.detail ?? ""}</p>
        </section>
      </main>
    </div>
  );
}