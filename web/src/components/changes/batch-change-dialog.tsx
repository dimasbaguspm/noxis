import { useRef, useState } from "react";
import { title } from "radash";
import { CheckCircle2, FileUp, Plus, XCircle } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { useConnectionSelection } from "@/hooks/use-connection";
import { api, type BatchFileResult } from "@/lib/api";
import { statementCount } from "@/lib/statements";

type Row = { name: string; query: string; rollback: string };

// BatchChangeDialog uploads several .sql files at once: one file is one
// statement and becomes one change, every change shares a batch id, and the
// report says which files verified.
export function BatchChangeDialog({ onSubmitted }: { onSubmitted?: () => void }) {
  const { connection, pick } = useConnectionSelection();
  const [open, setOpen] = useState(false);
  const [notes, setNotes] = useState("");
  const [rows, setRows] = useState<Row[]>([]);
  const [busy, setBusy] = useState(false);
  const [results, setResults] = useState<BatchFileResult[] | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  async function addFiles(files: FileList | null) {
    if (!files || files.length === 0) return;
    const added = await Promise.all(
      Array.from(files).map(async (file) => ({ name: file.name, query: await file.text(), rollback: "" })),
    );
    setRows((prev) => [...prev, ...added]);
    setResults(null);
  }

  function patch(i: number, next: Partial<Row>) {
    setRows((prev) => prev.map((row, j) => (j === i ? { ...row, ...next } : row)));
  }

  async function submit() {
    if (!connection) {
      toast.error("pick the instance");
      return;
    }
    const bad = rows.find((r) => statementCount(r.query) !== 1);
    if (bad) {
      toast.error(`${bad.name}: one file must hold exactly one statement`);
      return;
    }
    if (rows.some((r) => !r.rollback.trim())) {
      toast.error("every file needs a rollback query");
      return;
    }
    setBusy(true);
    try {
      const res = await api.changes.submitBatch({ connection, notes, files: rows });
      setResults(res.results);
      const ok = res.results.filter((r) => r.ok).length;
      toast.success(`batch ${res.batchId.slice(0, 8)}: ${ok}/${res.results.length} verified`);
      onSubmitted?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  function reset() {
    setNotes("");
    setRows([]);
    setResults(null);
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) reset();
      }}
    >
      <DialogTrigger render={<Button variant="outline" />}>Batch .sql files</DialogTrigger>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Batch .sql files</DialogTitle>
          <DialogDescription>
            One file is one statement and becomes one change. Every file shares a batch id and is verified on its own.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3">
          <div className="grid gap-1.5">
            <Label>
              {title("instance")} <Required />
            </Label>
            <ConnectionSelect value={connection} onChange={pick} className="w-full" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="batch-notes">{title("notes")}</Label>
            <Input
              id="batch-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="batch note (optional)"
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <input
              ref={fileInput}
              type="file"
              accept=".sql,text/plain"
              multiple
              className="hidden"
              onChange={(e) => {
                void addFiles(e.target.files);
                e.target.value = "";
              }}
            />
            <Button type="button" size="sm" variant="outline" onClick={() => fileInput.current?.click()}>
              <FileUp /> Attach .sql files
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={() => setRows((prev) => [...prev, { name: `change-${prev.length + 1}.sql`, query: "", rollback: "" }])}
            >
              <Plus /> Add row
            </Button>
            <span className="text-xs text-muted-foreground">{rows.length} file(s)</span>
          </div>

          {rows.map((row, i) => {
            const bad = row.query.trim() !== "" && statementCount(row.query) !== 1;
            return (
              <div key={`${row.name}-${i}`} className="grid gap-2 rounded-md border border-border p-3">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-xs">{row.name}</span>
                  {bad && <Badge variant="destructive">one statement only</Badge>}
                  <span className="flex-1" />
                  <Button type="button" size="xs" variant="ghost" onClick={() => setRows((prev) => prev.filter((_, j) => j !== i))}>
                    Remove
                  </Button>
                </div>
                <Textarea
                  value={row.query}
                  onChange={(e) => patch(i, { query: e.target.value })}
                  rows={3}
                  className="font-mono text-xs"
                  placeholder="original query (one statement)"
                />
                <Textarea
                  value={row.rollback}
                  onChange={(e) => patch(i, { rollback: e.target.value })}
                  rows={2}
                  className="font-mono text-xs"
                  placeholder="rollback query"
                />
              </div>
            );
          })}

          {results && (
            <div className="grid gap-1 rounded-md border border-border p-3">
              {results.map((r) => (
                <p key={r.name} className="flex items-start gap-2 text-xs">
                  {r.ok ? (
                    <CheckCircle2 className="mt-0.5 size-3.5 shrink-0 text-emerald-500" />
                  ) : (
                    <XCircle className="mt-0.5 size-3.5 shrink-0 text-destructive" />
                  )}
                  <span className={r.ok ? "text-muted-foreground" : "text-destructive"}>
                    <span className="font-medium">{r.name}</span>: {r.ok ? "verified" : r.error ?? "failed"}
                  </span>
                </p>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button type="button" onClick={() => void submit()} disabled={busy || rows.length === 0}>
            Submit batch
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Required() {
  return <span className="text-destructive">*</span>;
}
