import { useState } from "react";
import { title } from "radash";
import { CheckCircle2, PlugZap, XCircle } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlFileInput } from "@/components/changes/sql-file-input";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { classify } from "@/lib/statements";

type Input = {
  connection: string;
  notes: string;
  query: string;
  rollback: string;
};
type Outcome =
  | { id?: string; status?: string; checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[] }
  | undefined;

// CreateChangeForm is the inline "new change" form: instance, notes, original
// query and rollback, then submit. No dialog — the form lives on the queue.
export function CreateChangeForm({ kind, onSubmit }: { kind: "dml" | "ddl"; onSubmit: (input: Input) => Promise<Outcome> }) {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const [notes, setNotes] = useState("");
  const [query, setQuery] = useState("");
  const [rollback, setRollback] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<{ target?: string; ok?: boolean; detail?: string; error?: string }[]>([]);

  const detected = classify(query);

  async function submit() {
    if (!connection) {
      toast.error("pick the instance this change targets");
      return;
    }
    if (!query.trim() || !rollback.trim()) {
      toast.error("query and rollback are required");
      return;
    }
    setBusy(true);
    const change = await onSubmit({
      connection,
      notes: notes.trim() || `manual ${kind} change`,
      query,
      rollback,
    });
    setBusy(false);
    if (!change) return;
    if (change.status === "failed" || !change.id) {
      setFailure(change.checks ?? []);
      toast.error("validation failed — nothing was submitted");
      return;
    }
    setFailure([]);
    setNotes("");
    setQuery("");
    setRollback("");
    toast.success(`submitted for review · ${change.id.slice(0, 8)}`);
  }

  return (
    <Card>
      <CardContent className="grid gap-3 pt-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label>
              {title("instance")} <Required />
            </Label>
            <ConnectionSelect value={connection} onChange={pick} className="w-full" />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor="create-notes">{title("notes")}</Label>
            <Input
              id="create-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="what and why (optional)"
            />
          </div>
        </div>

        <div className="grid gap-1.5">
          <Label className="flex items-center gap-2">
            {title("original query")} <Required />
            {query.trim() && <Badge variant="secondary">{title(detected)}</Badge>}
            <span className="flex-1" />
            <SqlFileInput onLoad={(sql) => setQuery(sql)} />
          </Label>
          <SqlEditor value={query} onChange={setQuery} tables={tables} height="10rem" />
        </div>

        <div className="grid gap-1.5">
          <Label className="flex items-center gap-2">
            {title("rollback query")} <Required />
            <span className="flex-1" />
            <SqlFileInput onLoad={(sql) => setRollback(sql)} />
          </Label>
          <SqlEditor value={rollback} onChange={setRollback} tables={tables} height="8rem" />
        </div>

        {failure.length > 0 && (
          <div className="grid gap-1 rounded-md border border-destructive/40 p-3">
            {failure.map((check) => (
              <p key={check.target} className="flex items-start gap-2 text-xs">
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

        <div className="flex items-center justify-between gap-2">
          <p className="text-xs text-muted-foreground">
            The query and its rollback are dry-run together before anything is recorded.
          </p>
          <Button onClick={() => void submit()} disabled={busy}>
            <PlugZap /> Submit for review
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function Required() {
  return <span className="text-destructive">*</span>;
}