import { useChangeSubmit, type ChangeSubmitInitial } from "@/hooks/use-change-submit";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
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
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { SqlEditor } from "@/components/sql/sql-editor";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { classify } from "@/lib/statements";
import { P } from "@/components/ui/typography";

const submitSchema = z.object({
  notes: z.string().trim().min(1, "Notes are required"),
  query: z.string().min(1, "Original query is required"),
  rollback: z.string().min(1, "Rollback query is required"),
});

type FormValues = z.infer<typeof submitSchema>;

// ChangeSubmitDialog closes after recording; validation details belong on the change page.
export type { ChangeSubmitInitial };

export function ChangeSubmitDialog({
  kind,
  onSubmit,
  initial,
  trigger,
}: {
  kind: "dml" | "ddl";
  onSubmit: (input: {
    driver: string;
    connection: string;
    notes: string;
    query: string;
    rollback: string;
  }) => Promise<
    | {
        id?: string;
        status?: string;
        checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[];
      }
    | undefined
  >;
  // initial prefills the form (raising the inverse of a reviewed change); trigger replaces the default "Create ..." button.
  initial?: ChangeSubmitInitial;
  trigger?: React.ReactElement;
}) {
  // the instance is part of the change, so it is chosen here, not on the page
  const { connection: picked, driver, connections, pick } = useDriverConnections();
  // raising the inverse of a reviewed change starts on that change's instance
  const connection = initial?.connection ?? picked;
  const { data: tables } = useConnectionSchemaQuery(driver, connection); // completion follows the chosen instance
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    defaultValues: { notes: "", query: "", rollback: "" },
    resolver: zodResolver(submitSchema),
  });
  const { open, busy, failure, setDialogOpen, setBusy, setFailure } = useChangeSubmit({
    initial,
    reset,
  });

  const onValid = handleSubmit(async (values) => {
    if (!driver || !connection) {
      toast.error("Pick the instance this change targets");
      return;
    }
    setBusy(true);
    const change = await onSubmit({
      driver,
      connection,
      notes: values.notes,
      query: values.query,
      rollback: values.rollback,
    });
    setBusy(false);
    if (!change) return;
    if (change.status === "failed" || !change.id) {
      // rejected proposals are not recorded: stay open and show why
      setFailure(change.checks ?? []);
      toast.error("Validation failed — nothing was submitted");
      return;
    }
    toast.success(`Submitted for review · ${change.id.slice(0, 8)}`);
    setFailure([]);
    reset();
    setDialogOpen(false);
  });

  const detected = classify(watch("query"));

  return (
    <Sheet open={open} onOpenChange={setDialogOpen}>
      {trigger ? (
        <SheetTrigger render={trigger} />
      ) : (
        <SheetTrigger render={<Button />}>Create {kind.toUpperCase()} change</SheetTrigger>
      )}
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Submit {kind.toUpperCase()} change</SheetTitle>
          <SheetDescription>
            Dry-run together; nothing changes until this is approved and applied.
          </SheetDescription>
        </SheetHeader>

        <SheetBody>
          <form id="submit-change" onSubmit={onValid} className="grid gap-4">
            <FieldGroup>
              <Field>
                <FieldLabel>
                  Instance <Required />
                </FieldLabel>
                <ConnectionSelect
                  connections={connections}
                  value={connection}
                  onChange={pick}
                  className="w-full"
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="change-notes">
                  Notes <Required />
                </FieldLabel>
                <Input
                  id="change-notes"
                  placeholder="What and why"
                  aria-invalid={!!errors.notes}
                  {...register("notes")}
                />
                <FieldError errors={[errors.notes]} />
              </Field>

              <Field>
                <FieldLabel className="flex items-center gap-2">
                  Query <Required />
                  {watch("query").trim() && (
                    <Badge variant={detected === kind ? "secondary" : "destructive"}>
                      {title(detected)}
                    </Badge>
                  )}
                </FieldLabel>
                <SqlEditor
                  value={watch("query")}
                  onChange={(sql) => setValue("query", sql)}
                  tables={tables}
                  height="12rem"
                />
                <input type="hidden" aria-hidden {...register("query")} />
                <FieldError errors={[errors.query]} />
              </Field>

              <Field>
                <FieldLabel>
                  Rollback <Required />
                </FieldLabel>
                <SqlEditor
                  value={watch("rollback")}
                  onChange={(sql) => setValue("rollback", sql)}
                  tables={tables}
                  height="9rem"
                />
                <input type="hidden" aria-hidden {...register("rollback")} />
                <FieldError errors={[errors.rollback]} />
              </Field>
            </FieldGroup>
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
          </form>
        </SheetBody>

        <SheetFooter className="items-center">
          <P className="mr-auto">Validated on submit against the selected instance.</P>
          <Button type="submit" form="submit-change" disabled={busy}>
            Submit for review
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function Required() {
  return <span className="text-destructive">*</span>;
}
