import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { title } from "radash";
import { CheckCircle2, XCircle } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetBody,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Field, FieldLabel, FieldLayout } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { SqlEditor } from "@/components/sql/sql-editor";
import { useConnectionSelection } from "@/hooks/use-connection";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { classify } from "@/lib/statements";

type FormValues = { notes: string; query: string; rollback: string };

// ChangeSubmitDialog collects {notes, query, rollback} and closes as soon as
// the change is recorded — validation results live on the detail page, so the
// modal never lingers. react-hook-form owns the fields; the statement editors
// are wired by setValue because they are custom components, not inputs.
export type ChangeSubmitInitial = {
  connection?: string;
  notes?: string;
  query?: string;
  rollback?: string;
};

export function ChangeSubmitDialog({
  kind,
  onSubmit,
  initial,
  trigger,
}: {
  kind: "dml" | "ddl";
  onSubmit: (input: { connection: string; notes: string; query: string; rollback: string }) => Promise<
    { id?: string; status?: string; checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[] } | undefined
  >;
  // initial prefills the form (raising the inverse of a reviewed change);
  // trigger replaces the default "Create ..." button.
  initial?: ChangeSubmitInitial;
  trigger?: React.ReactElement;
}) {
  // the instance is part of the change, so it is chosen here, not on the page
  const { connection, pick } = useConnectionSelection(initial?.connection);
  const { data: tables } = useConnectionSchemaQuery(connection); // completion follows the chosen instance
  const { register, handleSubmit, watch, setValue, reset } = useForm<FormValues>({
    defaultValues: { notes: "", query: "", rollback: "" },
  });
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<{ target?: string; ok?: boolean; detail?: string; error?: string }[]>([]);

  // seed the form when the dialog opens (the initial may change after a refresh)
  useEffect(() => {
    if (open) reset({ notes: initial?.notes ?? "", query: initial?.query ?? "", rollback: initial?.rollback ?? "" });
  }, [open, initial, reset]);

  const onValid = handleSubmit(async (values) => {
    if (!connection) {
      toast.error("pick the instance this change targets");
      return;
    }
    if (!values.notes.trim() || !values.query.trim() || !values.rollback.trim()) {
      toast.error("notes, query and rollback are all required");
      return;
    }
    setBusy(true);
    const change = await onSubmit({ connection, notes: values.notes, query: values.query, rollback: values.rollback });
    setBusy(false);
    if (!change) return;
    if (change.status === "failed" || !change.id) {
      // rejected proposals are not recorded: stay open and show why
      setFailure(change.checks ?? []);
      toast.error("validation failed — nothing was submitted");
      return;
    }
    toast.success(`submitted for review · ${change.id.slice(0, 8)}`);
    setFailure([]);
    reset();
    setOpen(false);
  });

  const detected = classify(watch("query"));

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      {trigger ? (
        <SheetTrigger render={trigger} />
      ) : (
        <SheetTrigger render={<Button />}>Create {kind.toUpperCase()} change</SheetTrigger>
      )}
      <SheetContent className="sm:max-w-3xl">
        <SheetHeader>
          <SheetTitle>Submit {kind.toUpperCase()} change</SheetTitle>
          <SheetDescription>
            The query and its rollback are dry-run together in a transaction that always rolls back —
            nothing changes until a reviewer approves and applies it.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={onValid} className="flex flex-1 flex-col">
          <SheetBody className="grid gap-4">
          <FieldLayout>
          <Field>
            <FieldLabel>
              {title("instance")} <Required />
            </FieldLabel>
            <ConnectionSelect value={connection} onChange={pick} className="w-full" />
          </Field>
          <Field>
            <FieldLabel htmlFor="change-notes">
              {title("notes")} <Required />
            </FieldLabel>
            <Input id="change-notes" placeholder="what and why" {...register("notes")} />
          </Field>

          <Field>
            <FieldLabel className="flex items-center gap-2">
              {title("original query")} <Required />
              {watch("query").trim() && (
                <Badge variant={detected === kind ? "secondary" : "destructive"}>{title(detected)}</Badge>
              )}
            </FieldLabel>
            <SqlEditor value={watch("query")} onChange={(sql) => setValue("query", sql)} tables={tables} height="12rem" />
          </Field>

          <Field>
            <FieldLabel>
              {title("rollback query")} <Required />
            </FieldLabel>
            <SqlEditor value={watch("rollback")} onChange={(sql) => setValue("rollback", sql)} tables={tables} height="9rem" />
          </Field>
          </FieldLayout>
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
          </SheetBody>

          <SheetFooter className="items-center">
            <p className="mr-auto text-xs text-muted-foreground">
              validation runs on submit: syntax and dry run against the instance
            </p>
            <Button type="submit" disabled={busy}>
              Submit for review
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  );
}

function Required() {
  return <span className="text-destructive">*</span>;
}