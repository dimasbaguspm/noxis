import { useChangeSheet } from "@/hooks/use-change-sheet";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { title } from "radash";
import { PlugZap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlFileInput } from "@/components/changes/sql-file-input";
import { useConnectionSelection } from "@/hooks/use-connection";
import { CAPABILITY } from "@/lib/console-nav";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { classify } from "@/lib/statements";

// Choice is one submitted change plus the driver of the instance it targets: the engine path is the driver, not the console's grouping.
type Choice = { driver: string; connection: string; notes: string; query: string; rollback: string };
const createSchema = z.object({
  notes: z.string().optional(),
  query: z.string().min(1, "Original query is required"),
  rollback: z.string().min(1, "Rollback query is required"),
});

type FormValues = z.infer<typeof createSchema>;
type Outcome =
  | {
      id?: string;
      status?: string;
      checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[];
    }
  | undefined;

// CreateChangeSheet is the "new change" form: one roomy Sheet asking only for the instance, notes, the original query and its rollback.
export function CreateChangeSheet({
  kind,
  onSubmit,
}: {
  kind: "dml" | "ddl";
  onSubmit: (input: Choice) => Promise<Outcome>;
}) {
  const { connection, driver, pick } = useConnectionSelection();
  const { data: tables } = useConnectionSchemaQuery(driver, connection);
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    defaultValues: { notes: "", query: "", rollback: "" },
    resolver: zodResolver(createSchema),
  });
  const { open, busy, serverError, setSheetOpen, setBusy, setServerError } = useChangeSheet({
    reset,
  });

  const detected = classify(watch("query"));

  const onValid = handleSubmit(async (values) => {
    if (!driver || !connection) {
      toast.error("Pick the instance this change targets");
      return;
    }
    setBusy(true);
    const change = await onSubmit({
      driver,
      connection,
      notes: (values.notes ?? "").trim() || `manual ${kind} change`,
      query: values.query,
      rollback: values.rollback,
    });
    setBusy(false);
    if (!change) return;
    if (change.status === "failed" || !change.id) {
      setServerError(change.checks?.find((c) => !c.ok)?.error ?? "validation failed");
      toast.error("Validation failed — nothing was submitted");
      return;
    }
    setServerError(undefined);
    toast.success(`Submitted for review · ${change.id.slice(0, 8)}`);
    setSheetOpen(false);
  });

  return (
    <Sheet open={open} onOpenChange={setSheetOpen}>
      <SheetTrigger render={<Button />}>Create</SheetTrigger>
      <SheetContent>
        <SheetHeader>
          <SheetTitle>New {kind.toUpperCase()} change</SheetTitle>
          <SheetDescription>
            Query and rollback are dry-run together before anything is recorded.
          </SheetDescription>
        </SheetHeader>

        <SheetBody>
          <form id="create-change" onSubmit={onValid} className="grid gap-4">
            <Field>
              <FieldLabel>Instance *</FieldLabel>
              <ConnectionSelect
                capability={CAPABILITY.SQL}
                value={connection}
                onChange={pick}
                className="w-full"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor="sheet-notes">Notes</FieldLabel>
              <Input id="sheet-notes" placeholder="Optional note" {...register("notes")} />
            </Field>

            <Field>
              <FieldLabel className="flex items-center gap-2">
                Query *
                {watch("query").trim() && <Badge variant="secondary">{title(detected)}</Badge>}
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setValue("query", sql)} />
              </FieldLabel>
              <SqlEditor
                value={watch("query")}
                onChange={(sql) => setValue("query", sql)}
                tables={tables}
                height="12rem"
              />
              <input type="hidden" aria-hidden {...register("query")} />
              <FieldError>{serverError ?? errors.query?.message}</FieldError>
            </Field>

            <Field>
              <FieldLabel className="flex items-center gap-2">
                Rollback *
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setValue("rollback", sql)} />
              </FieldLabel>
              <SqlEditor
                value={watch("rollback")}
                onChange={(sql) => setValue("rollback", sql)}
                tables={tables}
                height="10rem"
              />
              <input type="hidden" aria-hidden {...register("rollback")} />
              <FieldError errors={[errors.rollback]} />
            </Field>
          </form>
        </SheetBody>

        <SheetFooter>
          <Button type="submit" form="create-change" disabled={busy}>
            <PlugZap /> Submit
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
