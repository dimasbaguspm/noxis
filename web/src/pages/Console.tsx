import { useEffect, useState } from "react";
import Changes from "../components/Changes";
import Grid from "../components/Grid";
import {
  DRIVERS,
  deleteConnection,
  getRows,
  listChanges,
  type Change,
  type Connection,
  type Result,
  runOp,
  saveConnection,
  testConnection,
} from "../api/client";
import { useStore } from "../store";

const EMPTY: Connection = { name: "", driver: "postgres", host: "", port: 5432, user: "", password: "", database: "" };

export default function Console() {
  const { connections, active, schema, refresh, pick, busy, setBusy } = useStore();
  const [form, setForm] = useState<Connection>(EMPTY);
  const [showForm, setShowForm] = useState(false);
  const [msg, setMsg] = useState("");
  const [sql, setSql] = useState("SELECT 1");
  const [result, setResult] = useState<Result | null>(null);
  const [resultErr, setResultErr] = useState("");
  const [changes, setChanges] = useState<Change[]>([]);

  useEffect(() => {
    void refresh();
    void loadChanges();
  }, [refresh]);

  async function loadChanges() {
    const res = await listChanges();
    setChanges(res.data?.changes ?? []);
  }

  async function addConnection() {
    if (!form.name.trim() || !form.host.trim() || !form.user.trim() || !form.database.trim()) return;
    setBusy(true);
    const res = await saveConnection({ ...form, port: Number(form.port) || 5432 });
    setBusy(false);
    if (!res.ok) {
      setMsg(String((res.data as unknown as { error?: string })?.error ?? "save failed"));
      return;
    }
    setForm(EMPTY);
    setShowForm(false);
    await refresh();
    pick(form.name);
  }

  async function test(name: string) {
    setBusy(true);
    const res = await testConnection(name);
    setBusy(false);
    setMsg(res.ok ? `${name}: ok` : `${name}: ${String((res.data as unknown as { error?: string })?.error ?? "failed")}`);
  }

  async function remove(name: string) {
    await deleteConnection(name);
    await refresh();
  }

  async function preview(schemaName: string, table: string) {
    setResultErr("");
    const res = await getRows(active, schemaName, table, 100);
    if (!res.ok) {
      setResult(null);
      setResultErr(String((res.data as unknown as { error?: string })?.error ?? "failed"));
      return;
    }
    setResult(res.data);
  }

  async function read() {
    if (!active || !sql.trim()) return;
    setBusy(true);
    const res = await runOp(active, "query", { sql: sql.trim() });
    setBusy(false);
    if (!res.ok) {
      setResult(null);
      setResultErr(String((res.data as unknown as { error?: string })?.error ?? "failed"));
      return;
    }
    setResultErr("");
    setResult(res.data);
  }

  return (
    <div className="min-h-screen bg-[#0d1117] text-[#c9d1d9]">
      <header className="flex flex-wrap items-center gap-3 border-b border-[#30363d] bg-[#161b22] px-4 py-2.5">
        <h1 className="m-0 text-[15px] font-semibold text-[#58a6ff]">noxis</h1>
        <div className="flex flex-wrap gap-1.5">
          {connections.map((c) => (
            <button key={c.name} onClick={() => pick(c.name)}
              className={`rounded-full border px-3 py-1 text-sm ${c.name === active ? "border-[#58a6ff] text-[#58a6ff]" : "border-[#30363d] bg-[#0d1117] hover:border-[#58a6ff]"}`}>
              {c.name}
            </button>
          ))}
        </div>
        <span className="flex-1" />
        <span className="text-xs text-[#8b949e]">{msg}</span>
        <button onClick={() => setShowForm((v) => !v)} className="rounded-md border border-[#30363d] px-3 py-1 text-sm">
          {showForm ? "cancel" : "+ connection"}
        </button>
      </header>

      {showForm && (
        <div className="border-b border-[#30363d] bg-[#161b22] px-4 py-3">
          <div className="mx-auto grid max-w-[1400px] gap-2 md:grid-cols-6">
            <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="name"
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <select value={form.driver} onChange={(e) => setForm({ ...form, driver: e.target.value, port: e.target.value === "mysql" ? 3306 : 5432 })}
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]">
              {DRIVERS.map((d) => <option key={d}>{d}</option>)}
            </select>
            <input value={form.host} onChange={(e) => setForm({ ...form, host: e.target.value })} placeholder="host"
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <input value={form.port} onChange={(e) => setForm({ ...form, port: Number(e.target.value) })} placeholder="port" type="number"
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <input value={form.user} onChange={(e) => setForm({ ...form, user: e.target.value })} placeholder="user"
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <input value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="password" type="password"
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <input value={form.database} onChange={(e) => setForm({ ...form, database: e.target.value })} placeholder="database"
              className="rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
            <button onClick={() => void addConnection()} disabled={busy}
              className="rounded-md bg-[#1f6feb] px-4 py-1.5 text-white disabled:opacity-50">save</button>
          </div>
        </div>
      )}

      <main className="mx-auto grid max-w-[1400px] gap-4 p-4 lg:grid-cols-[300px_1fr_380px]">
        <div className="space-y-4">
          <section className="rounded-lg border border-[#30363d] bg-[#161b22] p-4">
            <h2 className="mb-2.5 text-[13px] uppercase tracking-wider text-[#8b949e]">connections</h2>
            {connections.length === 0 && <p className="text-xs text-[#8b949e]">none yet — add one above</p>}
            {connections.map((c) => (
              <div key={c.name} className={`mb-1.5 rounded-md border p-2 ${c.name === active ? "border-[#58a6ff]/50" : "border-[#30363d]"}`}>
                <div className="flex items-center gap-2">
                  <span className="text-[13px]">{c.name}</span>
                  <span className="flex-1" />
                  <span className="text-[11px] text-[#8b949e]">{c.driver}</span>
                </div>
                <p className="text-[11px] text-[#8b949e]">{c.user}@{c.host}:{c.port}/{c.database}</p>
                <div className="mt-1 flex gap-1.5">
                  <button className="rounded border border-[#30363d] px-2 py-0.5 text-xs" onClick={() => void test(c.name ?? "")}>test</button>
                  <button className="rounded border border-[#f85149]/50 px-2 py-0.5 text-xs text-[#f85149]" onClick={() => void remove(c.name ?? "")}>delete</button>
                </div>
              </div>
            ))}
          </section>

          <section className="rounded-lg border border-[#30363d] bg-[#161b22] p-4">
            <h2 className="mb-2.5 text-[13px] uppercase tracking-wider text-[#8b949e]">schema — {active}</h2>
            {schema.length === 0 && <p className="text-xs text-[#8b949e]">no tables</p>}
            <div className="max-h-[320px] space-y-0.5 overflow-auto">
              {schema.map((t) => (
                <button key={`${t.schema}.${t.name}`} onClick={() => void preview(t.schema, t.name)}
                  className="block w-full rounded px-2 py-1 text-left text-[13px] hover:bg-[#21262d]">
                  <span className="text-[#8b949e]">{t.schema}.</span>{t.name}
                  <span className="ml-1 text-[11px] text-[#8b949e]">({t.columns.length})</span>
                </button>
              ))}
            </div>
          </section>
        </div>

        <section className="rounded-lg border border-[#30363d] bg-[#161b22] p-4">
          <h2 className="mb-2.5 text-[13px] uppercase tracking-wider text-[#8b949e]">read — {active}</h2>
          <textarea value={sql} onChange={(e) => setSql(e.target.value)}
            onKeyDown={(e) => { if ((e.metaKey || e.ctrlKey) && e.key === "Enter") void read(); }}
            className="min-h-[110px] w-full rounded-md border border-[#30363d] bg-[#0d1117] px-3 py-2 font-mono outline-none focus:border-[#58a6ff]" />
          <button onClick={() => void read()} disabled={busy || !active}
            className="mt-2 rounded-md bg-[#1f6feb] px-4 py-1.5 text-white disabled:opacity-50">
            run (⌘↵)
          </button>
          <p className="mb-2 mt-2 text-xs text-[#8b949e]">
            SELECT / EXPLAIN run directly — writes go through the change workflow
          </p>
          <Grid result={result} error={resultErr} />
        </section>

        <Changes changes={changes} reload={loadChanges} />
      </main>
    </div>
  );
}