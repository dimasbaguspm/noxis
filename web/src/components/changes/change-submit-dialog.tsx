import { useState } from "react";
import { CheckCircle2, XCircle } from "lucide-react";
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
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useApp } from "@/app/app-provider";
import { classify } from "@/lib/statements";
import type { Change } from "@/lib/api";

// ChangeSubmitDialog collects {notes, query, rollback}, submits it for
// validation and shows the dry-run verdict without leaving the dialog.
export function ChangeSubmitDialog({
  kind,
  onSubmit,
}: {
  kind: "dml" | "ddl";
  onSubmit: (input: { connection: string; notes: string; query: string; rollback: string }) => Promise<Change | undefined>;
}) {
  const { active } = useApp();
  const [open, setOpen] = useState(false);
  const [notes, setNotes] = useState("");
  const [query, setQuery] = useState("");
  const [rollback, setRollback] = useState("");
  const [result, setResult] = useState<Change | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit() {
    if (!active || !notes.trim() || !query.trim() || !rollback.trim()) {
      toast.error("connection, notes, query and rollback are all required");
      return;
    }
    setBusy(true);
    const change = await onSubmit({ connection: active, notes, query, rollback });
    setBusy(false);
    if (!change) return;
    setResult(change);
    if (change.status === "validated") toast.success("validated — ready for approval");
    else if (change.status === "unverified") toast.warning("submitted unverified (driver cannot dry-run DDL)");
    else toast.error("validation failed");
  }

  function reset(next: boolean) {
    setOpen(next);
    if (!next) {
      setNotes("");
      setQuery("");
      setRollback("");
      setResult(null);
    }
  }

  return (
    <Dialog open={open} onOpenChange={reset}>
      <DialogTrigger render={<Button />}>New {kind.toUpperCase()} change</DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Submit {kind.toUpperCase()} change</DialogTitle>
          <DialogDescription>
            The query and its rollback are dry-run together in a transaction that always rolls back —
            nothing changes until you approve and apply.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3">
          <div className="grid gap-1.5">
            <Label>Notes</Label>
            <Textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="what and why"
              className="min-h-16"
            />
          </div>
          <div className="grid gap-1.5">
            <Label className="flex items-center gap-2">
              Query
              {query.trim() && <Badge variant="secondary">{classify(query).toUpperCase()}</Badge>}
            </Label>
            <Textarea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={kind === "ddl" ? "ALTER TABLE …" : "INSERT INTO … / UPDATE … / DELETE …"}
              className="min-h-28 font-mono"
            />
          </div>
          <div className="grid gap-1.5">
            <Label>Rollback</Label>
            <Textarea
              value={rollback}
              onChange={(e) => setRollback(e.target.value)}
              placeholder="the SQL that undoes it"
              className="min-h-24 font-mono"
            />
          </div>

          {result && (
            <div className="grid gap-1 rounded-md border border-border p-3">
              {(result.checks ?? []).map((check) => (
                <p key={`${check.target}:${check.error ?? check.detail ?? ""}`} className="flex items-start gap-2 text-xs">
                  {check.ok ? (
                    <CheckCircle2 className="mt-0.5 size-3.5 shrink-0 text-emerald-500" />
                  ) : (
                    <XCircle className="mt-0.5 size-3.5 shrink-0 text-destructive" />
                  )}
                  <span className={check.ok ? "text-muted-foreground" : "text-destructive"}>
                    <span className="font-medium">{check.target}</span>
                    {check.ok ? `: ${check.detail ?? "ok"}` : `: ${check.error ?? "failed"}`}
                  </span>
                </p>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button onClick={() => void submit()} disabled={busy || !active}>
            Submit
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
