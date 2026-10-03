import { useState } from "react";
import {
  approveChange,
  applyChange,
  listChanges,
  rejectChange,
  submitChange,
  type Change,
} from "../api/client";
import { useStore } from "../store";

const STATUS_STYLE: Record<string, string> = {
  validated: "text-[#58a6ff] border-[#58a6ff]/40",
  unverified: "text-[#d29922] border-[#d29922]/40",
  approved: "text-[#d29922] border-[#d29922]/40",
  applied: "text-[#3fb950] border-[#3fb950]/40",
  failed: "text-[#f85149] border-[#f85149]/40",
};

// The change workflow pane: submit (query + rollback + notes) → validated or
// failed → approve (self-approval is fine) → apply.
export default function Changes({ changes, reload }: { changes: Change[]; reload: () => Promise<void> }) {
  const { active, setBusy } = useStore();
  const [notes, setNotes] = useState("");
  const [query, setQuery] = useState("");
  const [rollback, setRollback] = useState("");
  const [open, setOpen] = useState<string>("");

  async function submit() {
    if (!active || !notes.trim() || !query.trim() || !rollback.trim()) return;
    setBusy(true);
    await submitChange({ connection: active, notes: notes.trim(), query: query.trim(), rollback: rollback.trim() });
    setBusy(false);
    setNotes("");
    setQuery("");
    setRollback("");
    await reload();
  }

  async function act(id: string, fn: (id: string) => Promise<unknown>) {
    setBusy(true);
    await fn(id);
    setBusy(false);
    await reload();
  }

  return (
    <section className="rounded-lg border border-[#30363d] bg-[#161b22] p-4">
      <h2 className="mb-2.5 text-[13px] uppercase tracking-wider text-[#8b949e]">changes — {active}</h2>
      <div className="space-y-2">
        <input value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="notes — what and why"
          className="w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 outline-none focus:border-[#58a6ff]" />
        <textarea value={query} onChange={(e) => setQuery(e.target.value)} placeholder="query — the change (DML/DDL)"
          className="min-h-[90px] w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 font-mono outline-none focus:border-[#58a6ff]" />
        <textarea value={rollback} onChange={(e) => setRollback(e.target.value)} placeholder="rollback — the SQL that undoes it"
          className="min-h-[60px] w-full rounded-md border border-[#30363d] bg-[#0d1117] px-2 py-1.5 font-mono outline-none focus:border-[#58a6ff]" />
        <button onClick={() => void submit()} disabled={!active}
          className="rounded-md bg-[#1f6feb] px-4 py-1.5 text-white disabled:opacity-50">
          submit &amp; validate
        </button>
        <p className="text-xs text-[#8b949e]">both statements are dry-run in a transaction — nothing persists on submit</p>
      </div>

      <div className="mt-4 space-y-2">
        {changes.length === 0 && <p className="text-xs text-[#8b949e]">no changes yet</p>}
        {changes.map((c) => (
          <div key={c.id} className="rounded-md border border-[#30363d] bg-[#0d1117] p-2.5">
            <div className="flex items-center gap-2">
              <span className={`rounded border px-1.5 text-[11px] uppercase ${STATUS_STYLE[c.status ?? ""] ?? ""}`}>
                {c.status}
              </span>
              <span className="text-[13px]">{c.notes}</span>
              <span className="flex-1" />
              <span className="text-[11px] text-[#8b949e]">{c.connection}</span>
            </div>
            <div className="mt-1 flex gap-1.5">
              <button className="rounded border border-[#30363d] px-2 py-0.5 text-xs"
                onClick={() => setOpen(open === c.id ? "" : (c.id ?? ""))}>
                {open === c.id ? "hide" : "details"}
              </button>
              {(c.status === "validated" || c.status === "unverified") && (
                <button className="rounded border border-[#d29922]/50 px-2 py-0.5 text-xs text-[#d29922]"
                  onClick={() => c.id && act(c.id, approveChange)}>approve</button>
              )}
              {c.status === "approved" && (
                <button className="rounded border border-[#3fb950]/50 px-2 py-0.5 text-xs text-[#3fb950]"
                  onClick={() => c.id && act(c.id, applyChange)}>apply</button>
              )}
              <button className="rounded border border-[#f85149]/50 px-2 py-0.5 text-xs text-[#f85149]"
                onClick={() => c.id && act(c.id, rejectChange)}>discard</button>
            </div>
            {open === c.id && (
              <div className="mt-2 space-y-1 border-t border-[#21262d] pt-2">
                {(c.checks ?? []).map((k, i) => (
                  <p key={i} className={`text-xs ${k.ok ? "text-[#3fb950]" : "text-[#f85149]"}`}>
                    {k.ok ? "✓" : "✗"} {k.target}: {k.ok ? k.detail : k.error}
                  </p>
                ))}
                <pre className="overflow-auto rounded border border-[#21262d] p-2 text-xs text-[#8b949e]">{c.query}</pre>
                <pre className="overflow-auto rounded border border-[#21262d] p-2 text-xs text-[#8b949e]">{c.rollback}</pre>
              </div>
            )}
          </div>
        ))}
      </div>
      <button className="mt-3 rounded border border-[#30363d] px-2 py-1 text-xs" onClick={() => void listChanges().then(reload)}>
        refresh
      </button>
    </section>
  );
}