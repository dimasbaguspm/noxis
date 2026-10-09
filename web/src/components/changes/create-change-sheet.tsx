import { useEffect, useRef, useState } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { title } from "radash";
import { CheckCircle2, FileUp, PlugZap, Plus, XCircle } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { SqlEditor } from "@/components/sql/sql-editor";
import { SqlFileInput } from "@/components/changes/sql-file-input";
import { useConnectionSelection, useSchema } from "@/hooks/use-connection";
import { api, type BatchFileResult } from "@/lib/api";
import { classify, statementCount } from "@/lib/statements";

type Choice = { connection: string; notes: string; query: string; rollback: string };
type BatchRow = { name: string; query: string; rollback: string };
type FormValues = { notes: string; query: string; rollback: string; rows: BatchRow[] };
type Outcome =
  | { id?: string; status?: string; checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[] }
  | undefined;

// CreateChangeSheet is the single "new change" form: one roomy Sheet holds the
// instance, notes, original query and rollback, plus a .sql batch section.
// Attached files become batch rows that submit through the batch endpoint.
export function CreateChangeSheet({
  kind,
  onSubmit,
  onSubmitted,
}: {
  kind: "dml" | "ddl";
  onSubmit: (input: Choice) => Promise<Outcome>;
  onSubmitted?: () => void;
}) {
  const { connection, pick } = useConnectionSelection();
  const { tables } = useSchema(connection);
  const { register, handleSubmit, watch, setValue, reset, control } = useForm<FormValues>({
    defaultValues: { notes: "", query: "", rollback: "", rows: [] },
  });
  const { fields, append, remove } = useFieldArray({ control, name: "rows" });
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [burning, setBurning] = useState(false);
  const [failure, setFailure] = useState<{ target?: string; ok?: boolean; detail?: string; error?: string }[]>([]);
  const [results, setResults] = useState<BatchFileResult[] | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) reset({ notes: "", query: "", rollback: "", rows: [] });
  }, [open, reset]);

  const detected = classify(watch("query"));
  const rows = watch("rows");

  async function addFiles(files: FileList | null) {
    if (!files || files.length === 0) return;
    const added = await Promise.all(
      Array.from(files).map(async (file) => ({ name: file.name, query: await file.text(), rollback: "" })),
    );
    append(added);
    setResults(null);
  }

  const onValid = handleSubmit(async (values) => {
    if (!connection) {
      toast.error("pick the instance this change targets");
      return;
    }
    if (!values.query.trim() || !values.rollback.trim()) {
      toast.error("query and rollback are required");
      return;
    }
    setBusy(true);
    const change = await onSubmit({
      connection,
      notes: values.notes.trim() || `manual ${kind} change`,
      query: values.query,
      rollback: values.rollback,
    });
    setBusy(false);
    if (!change) return;
    if (change.status === "failed" || !change.id) {
      setFailure(change.checks ?? []);
      toast.error("validation failed — nothing was submitted");
      return;
    }
    setFailure([]);
    toast.success(`submitted for review · ${change.id.slice(0, 8)}`);
    setOpen(false);
  });

  async function submitBatch() {
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
    setBurning(true);
    try {
      const res = await api.changes.submitBatch({
        connection,
        notes: watch("notes"),
        files: rows,
      });
      setResults(res.results);
      const ok = res.results.filter((r) => r.ok).length;
      toast.success(`batch ${res.batchId.slice(0, 8)}: ${ok}/${res.results.length} verified`);
      onSubmitted?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setBurning(false);
    }
  }

  return (
    <Sheet open={open} onOpenChange={(next) => { setOpen(next); setResults(null); }}>
      <SheetTrigger render={<Button />}>Create {kind.toUpperCase()} change</SheetTrigger>
      <SheetContent className="max-w-2xl">
        <SheetHeader>
          <SheetTitle>New {kind.toUpperCase()} change</SheetTitle>
          <SheetDescription>
            The query and its rollback are dry-run together before anything is recorded. Attach .sql
            files to raise a batch on the same instance.
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={onValid} className="grid min-h-0 flex-1 grid-cols-2 gap-3">
          <div className="grid gap-3 self-start">
            <div className="grid gap-1.5">
              <Label>
                {title("instance")} <Required />
              </Label>
              <ConnectionSelect value={connection} onChange={pick} className="w-full" />
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor="sheet-notes">{title("notes")}</Label>
              <Input id="sheet-notes" placeholder="what and why (optional)" {...register("notes")} />
            </div>
            <div className="grid gap-1.5">
              <Label className="flex items-center gap-2">
                {title("original query")} <Required />
                {watch("query").trim() && <Badge variant="secondary">{title(detected)}</Badge>}
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setValue("query", sql)} />
              </Label>
              <SqlEditor value={watch("query")} onChange={(sql) => setValue("query", sql)} tables={tables} height="12rem" />
            </div>
            <div className="grid gap-1.5">
              <Label className="flex items-center gap-2">
                {title("rollback query")} <Required />
                <span className="flex-1" />
                <SqlFileInput onLoad={(sql) => setValue("rollback", sql)} />
              </Label>
              <SqlEditor value={watch("rollback")} onChange={(sql) => setValue("rollback", sql)} tables={tables} height="10rem" />
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
          </div>

          <div className="grid gap-3 self-start">
            <div className="flex items-center gap-2">
              <Label className="flex-1">{title("batch .sql files")}</Label>
              <span className="text-xs text-muted-foreground">{rows.length} file(s)</span>
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
                onClick={() => append({ name: `change-${rows.length + 1}.sql`, query: "", rollback: "" })}
              >
                <Plus /> Add row
              </Button>
            </div>

            {fields.map((field, i) => {
              const row = rows[i];
              const bad = row?.query.trim() !== "" && statementCount(row?.query ?? "") !== 1;
              return (
                <div key={field.id} className="grid gap-2 rounded-md border border-border p-3">
                  <input type="hidden" {...register(`rows.${i}.name`)} />
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-xs">{row?.name}</span>
                    {bad && <Badge variant="destructive">one statement only</Badge>}
                    <span className="flex-1" />
                    <Button type="button" size="xs" variant="ghost" onClick={() => remove(i)}>
                      Remove
                    </Button>
                  </div>
                  <Textarea
                    rows={3}
                    className="font-mono text-xs"
                    placeholder="original query (one statement)"
                    {...register(`rows.${i}.query`)}
                  />
                  <Textarea
                    rows={2}
                    className="font-mono text-xs"
                    placeholder="rollback query"
                    {...register(`rows.${i}.rollback`)}
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

          <SheetFooter className="col-span-2">
            <span className="mr-auto text-xs text-muted-foreground">one instance, one review queue</span>
            <Button type="button" variant="outline" disabled={burning || rows.length === 0} onClick={() => void submitBatch()}>
              <FileUp /> Submit batch
            </Button>
            <Button type="submit" disabled={busy}>
              <PlugZap /> Submit for review
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