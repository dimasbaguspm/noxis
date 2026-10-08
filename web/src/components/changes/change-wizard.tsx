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

const STEPS = ["Instance", "Kind", "Original Query", "Rollback Query", "Schedule"];

// ChangeWizard walks a change through instance → kind → query → rollback →
// schedule, one bounded step at a time. The step list is also the progress
// indicator, so a long form never hides what is left.
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
    scheduledAt?: number;
  }) => Promise<
    { id?: string; status?: string; checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[] } | undefined
  >;
}) {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState(0);
  const [notes, setNotes] = useState("");
  const [wizardKind, setWizardKind] = useState<Kind>(kind);
  const [query, setQuery] = useState("");
  const [rollback, setRollback] = useState("");
  const [when, setWhen] = useState<"now" | "later">("now");
  const [scheduledAt, setScheduledAt] = useState("");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<{ target?: string; ok?: boolean; detail?: string; error?: string }[]>([]);

  const detected = classify(query);
  const slot = scheduledAt ? Math.floor(new Date(scheduledAt).getTime() / 1000) : 0;
  const slotValid = when === "now" || (slot > Math.floor(Date.now() / 1000));
  const canAdvance = [Boolean(connection), Boolean(wizardKind), Boolean(query.trim()), Boolean(rollback.trim()), slotValid][step];

  function reset() {
    setStep(0);
    setNotes("");
    setQuery("");
    setRollback("");
    setWhen("now");
    setScheduledAt("");
    setFailure([]);
  }

  async function submit() {
    if (when === "later" && !slotValid) {
      toast.error("pick a time in the future");
      return;
    }
    setBusy(true);
    const change = await onSubmit({
      connection: connection ?? "",
      notes: notes.trim() || `manual ${wizardKind} change`,
      query,
      rollback,
      scheduledAt: when === "later" ? slot : 0,
    });
    setBusy(false);
    if (!change) return;
    if (change.status === "failed" || !change.id) {
      setFailure(change.checks ?? []);
      toast.error("validation failed — nothing was submitted");
      return;
    }
    toast.success(
      when === "later" ? `scheduled · ${change.id.slice(0, 8)}` : `submitted for review · ${change.id.slice(0, 8)}`,
    );
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
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>New {wizardKind.toUpperCase()} change</DialogTitle>
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

        <div className="grid min-h-56 gap-3">
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
            <div className="grid gap-2">
              <Label>{title("kind")}</Label>
              <div className="flex gap-2">
                {(["dml", "ddl"] as const).map((k) => (
                  <Button
                    key={k}
                    type="button"
                    variant={wizardKind === k ? "default" : "outline"}
                    onClick={() => setWizardKind(k)}
                  >
                    {k.toUpperCase()}
                  </Button>
                ))}
              </div>
              <p className="text-xs text-muted-foreground">
                The server re-derives the kind from the statement, so a mismatch is rejected at submit.
              </p>
            </div>
          )}

          {step === 2 && (
            <div className="grid gap-1.5">
              <Label className="flex items-center gap-2">
                {title("original query")} <Required />
                {query.trim() && (
                  <Badge variant={detected === wizardKind ? "secondary" : "destructive"}>{title(detected)}</Badge>
                )}
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setQuery(sql)} />
              </Label>
              <SqlEditor value={query} onChange={setQuery} tables={tables} height="14rem" />
            </div>
          )}

          {step === 3 && (
            <div className="grid gap-1.5">
              <Label className="flex items-center gap-2">
                {title("rollback query")} <Required />
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setRollback(sql)} />
              </Label>
              <SqlEditor value={rollback} onChange={setRollback} tables={tables} height="12rem" />
            </div>
          )}

          {step === 4 && (
            <div className="grid gap-3">
              <Label>{title("when")}</Label>
              <div className="flex gap-2">
                <Button type="button" variant={when === "now" ? "default" : "outline"} onClick={() => setWhen("now")}>
                  Execute immediately
                </Button>
                <Button type="button" variant={when === "later" ? "default" : "outline"} onClick={() => setWhen("later")}>
                  Execute after…
                </Button>
              </div>
              {when === "later" && (
                <div className="grid gap-1.5">
                  <Label htmlFor="wizard-when">{title("scheduled time")}</Label>
                  <Input
                    id="wizard-when"
                    type="datetime-local"
                    value={scheduledAt}
                    onChange={(e) => setScheduledAt(e.target.value)}
                  />
                  <p className="text-xs text-muted-foreground">
                    The scheduler applies the change once its slot passes; until then Apply stays refused.
                  </p>
                </div>
              )}
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
              {when === "later" ? "Schedule change" : "Submit for review"}
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
