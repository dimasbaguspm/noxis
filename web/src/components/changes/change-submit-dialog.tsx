import { useState } from "react";
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
import { ConnectionSelect } from "@/components/connection/connection-select";
import { SqlEditor } from "@/components/sql/sql-editor";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { classify } from "@/lib/statements";

// ChangeSubmitDialog collects {notes, query, rollback} and closes as soon as
// the change is recorded — validation results live on the detail page, so the
// modal never lingers.
export function ChangeSubmitDialog({
  kind,
  onSubmit,
}: {
  kind: "dml" | "ddl";
  onSubmit: (input: { connection: string; notes: string; query: string; rollback: string }) => Promise<
    { id?: string } | undefined
  >;
}) {
  // the instance is part of the change, so it is chosen here, not on the page
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection); // completion follows the chosen instance
  const [open, setOpen] = useState(false);
  const [notes, setNotes] = useState("");
  const [query, setQuery] = useState("");
  const [rollback, setRollback] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit() {
    if (!connection) {
      toast.error("pick the instance this change targets");
      return;
    }
    if (!notes.trim() || !query.trim() || !rollback.trim()) {
      toast.error("notes, query and rollback are all required");
      return;
    }
    setBusy(true);
    const change = await onSubmit({ connection, notes, query, rollback });
    setBusy(false);
    if (!change) return;
    toast.success(`submitted for review${change.id ? ` · ${change.id.slice(0, 8)}` : ""}`);
    setNotes("");
    setQuery("");
    setRollback("");
    setOpen(false);
  }

  const detected = classify(query);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>Create {kind.toUpperCase()} change</DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Submit {kind.toUpperCase()} change</DialogTitle>
          <DialogDescription>
            The query and its rollback are dry-run together in a transaction that always rolls back —
            nothing changes until a reviewer approves and applies it.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3">
          <div className="grid gap-1.5">
            <Label>
              Instance <Required />
            </Label>
            <ConnectionSelect value={connection} onChange={pick} className="w-full" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="change-notes">
              Notes <Required />
            </Label>
            <Input
              id="change-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="what and why"
            />
          </div>

          <div className="grid gap-1.5">
            <Label className="flex items-center gap-2">
              Query <Required />
              {query.trim() && <Badge variant={detected === kind ? "secondary" : "destructive"}>{detected}</Badge>}
            </Label>
            <SqlEditor value={query} onChange={setQuery} tables={tables} height="12rem" />
          </div>

          <div className="grid gap-1.5">
            <Label>
              Rollback <Required />
            </Label>
            <SqlEditor value={rollback} onChange={setRollback} tables={tables} height="9rem" />
          </div>
        </div>

        <DialogFooter className="items-center">
          <p className="mr-auto text-xs text-muted-foreground">
            validation runs on submit: syntax and dry run against the instance
          </p>
          <Button onClick={() => void submit()} disabled={busy}>
            Submit for review
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Required() {
  return <span className="text-destructive">*</span>;
}
