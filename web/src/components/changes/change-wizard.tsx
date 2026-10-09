import { useState } from "react";
import { title } from "radash";
import { CheckCircle2, ChevronLeft, ChevronRight, XCircle } from "lucide-react";
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
import { SqlFileInput } from "@/components/changes/sql-file-input";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { classify } from "@/lib/statements";

type Kind = "dml" | "ddl";

const STEPS = ["Instance", "Original Query", "Rollback Query"];

// ChangeWizard walks a change through instance → query → rollback. The kind is
// not asked for: the server derives it from the statement, and scheduling
// happens later, on the approved change.
export function ChangeWizard({
  kind,
  onSubmit,
}: {
  kind: Kind;
  onSubmit: (input: {
    connection: string;
    notes: string;
    query: string;
    rollback: string;
  }) => Promise<
    { id?: string; status?: string; checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[] } | undefined
  >;
}) {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState(0);
  const [notes, setNotes] = useState("");
  const [query, setQuery] = useState("");
  const [rollback, setRollback] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<{ target?: string; ok?: boolean; detail?: string; error?: string }[]>([]);

  const detected = classify(query);
  const canAdvance = [Boolean(connection), Boolean(query.trim()), Boolean(rollback.trim())][step];

  function reset() {
    setStep(0);
    setNotes("");
    setQuery("");
    setRollback("");
    setFailure([]);
  }

  async function submit() {
    setBusy(true);
    const change = await onSubmit({
      connection: connection ?? "",
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
    toast.success(`submitted for review · ${change.id.slice(0, 8)}`);
    reset();
    setOpen(false);
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) reset();
      }}
    >
      <DialogTrigger render={<Button />}>Create {kind.toUpperCase()} change</DialogTrigger>
      <DialogContent className="sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>New {kind.toUpperCase()} change</DialogTitle>
          <DialogDescription>
            Step {step + 1} of {STEPS.length} · {STEPS[step]}. The query and its rollback are dry-run
            together before anything is recorded.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-wrap items-center gap-1.5">
          {STEPS.map((name, i) => (
            <Badge key={name} variant={i === step ? "default" : i < step ? "secondary" : "outline"}>
              {i < step ? "✓ " : ""}
              {name}
            </Badge>
          ))}
        </div>

        <div className="grid min-h-72 gap-3">
          {step === 0 && (
            <>
              <div className="grid gap-1.5">
                <Label>
                  {title("instance")} <Required />
                </Label>
                <ConnectionSelect value={connection} onChange={pick} className="w-full" />
              </div>
              <div className="grid gap-1.5">
                <Label htmlFor="wizard-notes">{title("notes")}</Label>
                <Input
                  id="wizard-notes"
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="what and why (optional)"
                />
              </div>
            </>
          )}

          {step === 1 && (
            <div className="grid gap-1.5">
              <Label className="flex items-center gap-2">
                {title("original query")} <Required />
                {query.trim() && <Badge variant="secondary">{title(detected)}</Badge>}
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setQuery(sql)} />
              </Label>
              <SqlEditor value={query} onChange={setQuery} tables={tables} height="20rem" />
            </div>
          )}

          {step === 2 && (
            <div className="grid gap-1.5">
              <Label className="flex items-center gap-2">
                {title("rollback query")} <Required />
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setRollback(sql)} />
              </Label>
              <SqlEditor value={rollback} onChange={setRollback} tables={tables} height="20rem" />
            </div>
          )}
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

        <DialogFooter className="items-center">
          <Button
            type="button"
            variant="ghost"
            disabled={step === 0 || busy}
            onClick={() => setStep((s) => Math.max(0, s - 1))}
          >
            <ChevronLeft /> Back
          </Button>
          <span className="mr-auto text-xs text-muted-foreground">
            {STEPS[step]} · {step + 1}/{STEPS.length}
          </span>
          {step < STEPS.length - 1 ? (
            <Button type="button" disabled={!canAdvance} onClick={() => setStep((s) => s + 1)}>
              Next <ChevronRight />
            </Button>
          ) : (
            <Button type="button" disabled={busy || !canAdvance} onClick={() => void submit()}>
              Submit for review
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Required() {
  return <span className="text-destructive">*</span>;
}
