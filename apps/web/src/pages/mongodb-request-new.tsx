import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link, useNavigate } from "react-router-dom";
import { ChevronLeft, ChevronRight, CircleHelp, Send, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { JsonEditor } from "@/components/sql/json-editor";
import { useMongoCollectionsQuery, useMongoConnections, useMongoDatabasesQuery, useMongoSubmitChangeMutation } from "@/hooks/use-mongodb";
import type { Change } from "@/lib/api";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";
import { H2 } from "@/components/ui/typography";

// Write ops only: reads belong in Explore, and the queue refuses them.
const WRITES = [
  { value: "insert", label: "insert", hint: "a document, or an array when many" },
  { value: "update", label: "update", hint: "a filter plus an update document ($set, $inc…)" },
  { value: "delete", label: "delete", hint: "a filter is required" },
] as const;

const SAMPLES: Record<string, { filter: string; value: string }> = {
  insert: { filter: "", value: '{ "name": "example", "qty": 1 }' },
  update: { filter: '{ "name": "example" }', value: '{ "$set": { "qty": 2 } }' },
  delete: { filter: '{ "name": "example" }', value: "" },
};

function isJson(value: string): boolean {
  if (value.trim() === "") return true;
  try {
    JSON.parse(value);
    return true;
  } catch {
    return false;
  }
}

const requestSchema = z.object({
  connection: z.string().min(1, "Choose a connection"),
  op: z.enum(["insert", "update", "delete"]),
  notes: z.string().optional(),
  filter: z.string().refine(isJson, "Filter must be valid JSON"),
  value: z.string().refine(isJson, "Value must be valid JSON"),
  many: z.boolean(),
});
type RequestForm = z.infer<typeof requestSchema>;

// MongoRequestNewPage is the request wizard: pick the database, pick the collection, write the filter and the value, submit for review.
export default function MongoRequestNewPage() {
  const navigate = useNavigate();
  const { connections, connection, pick } = useMongoConnections();
  const { databases } = useMongoDatabasesQuery(connection);
  const [selectedDatabase, setSelectedDatabase] = useState("");
  const database = databases.some((item) => item.name === selectedDatabase) ? selectedDatabase : (databases[0]?.name ?? "");
  const { collections } = useMongoCollectionsQuery(connection, database);
  const [selectedCollection, setSelectedCollection] = useState("");
  const collection = collections.some((item) => item.name === selectedCollection) ? selectedCollection : (collections[0]?.name ?? "");
  const [step, setStep] = useState(1);
  const [failure, setFailure] = useState<Change>();
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    trigger,
    formState: { errors },
  } = useForm<RequestForm>({
    defaultValues: {
      connection: "",
      op: "update",
      notes: "",
      filter: SAMPLES.update?.filter ?? "",
      value: SAMPLES.update?.value ?? "",
      many: false,
    },
    resolver: zodResolver(requestSchema),
  });

  const opValue = watch("op");
  const submit = useMongoSubmitChangeMutation();

  const spec = WRITES.find((w) => w.value === opValue);

  function chooseOp(next: string) {
    const selected = next as RequestForm["op"];
    setValue("op", selected);
    setValue("filter", SAMPLES[selected]?.filter ?? "", { shouldValidate: false });
    setValue("value", SAMPLES[selected]?.value ?? "", { shouldValidate: false });
    setValue("many", false);
  }

  const onSend = handleSubmit(async (values) => {
    setFailure(undefined);
    if (!database || !collection) {
      toast.error("Choose a database and collection");
      return;
    }
    try {
      const change = (await submit.mutateAsync({
        connection: values.connection,
        notes: (values.notes ?? "").trim() || `${values.op} on ${database}.${collection}`,
        op: values.op,
        database,
        collection,
        filter: values.filter,
        value: values.value,
        many: values.many,
      })) as Change;
      if (change.status === "failed") {
        // a failed validation is a rejected proposal: it is not in the queue
        setFailure(change);
        toast.error("Validation failed — the request was not recorded");
        return;
      }
      toast.success(`Request ${change.status}: ${change.estAffected ?? 0} document(s) would be affected`);
      navigate(PAGE_ROUTES.MONGODB.REQUEST_DETAIL(change.id ?? ""));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  });

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-4">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" render={<Link to={PAGE_ROUTES.MONGODB.REQUESTS} />}>
          <ChevronLeft /> Requests
        </Button>
        <Badge variant="outline">step {step} of 3</Badge>
      </div>

      {step === 1 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Which connection and database?</CardTitle>
            <CardDescription>The request is validated against this database</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Field>
              <FieldLabel>Connection</FieldLabel>
              <Select
                value={watch("connection")}
                onValueChange={(v) => {
                  const name = String(v);
                  setValue("connection", name);
                  setSelectedDatabase("");
                  setSelectedCollection("");
                  pick(name);
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="No MongoDB registered" />
                </SelectTrigger>
                <SelectContent>
                  {connections.map((c) => (
                    <SelectItem key={c.name} value={c.name}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FieldError errors={[errors.connection]} />
            </Field>
            <Field>
              <FieldLabel>Database</FieldLabel>
              <Select
                value={database}
                onValueChange={(v) => {
                  setSelectedDatabase(String(v));
                  setSelectedCollection("");
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Pick a database" />
                </SelectTrigger>
                <SelectContent>
                  {databases.map((db) => (
                    <SelectItem key={db.name} value={db.name ?? ""}>
                      {db.name} · {db.collections ?? 0} collections
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

            </Field>
            <div className="flex justify-end">
              <Button
                onClick={async () => {
                  if (!(await trigger("connection"))) return;
                  if (!database) {
                    toast.error("Choose a database");
                    return;
                  }
                  setStep(2);
                }}
              >
                Next <ChevronRight />
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === 2 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Which collection, and what kind of write?</CardTitle>
            <CardDescription>
              {collection
                ? `${collections.find((c) => c.name === collection)?.count ?? 0} document(s) currently`
                : "pick the collection the request targets"}
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Field>
              <FieldLabel>Collection</FieldLabel>
              <Select value={collection} onValueChange={(v) => setSelectedCollection(String(v))}>
                <SelectTrigger>
                  <SelectValue placeholder="Pick a collection" />
                </SelectTrigger>
                <SelectContent>
                  {collections.map((c) => (
                    <SelectItem key={c.name} value={c.name ?? ""}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

            </Field>
            <Field>
              <FieldLabel>Operation</FieldLabel>
              <Select value={opValue} onValueChange={(v) => chooseOp(String(v))}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {WRITES.map((w) => (
                    <SelectItem key={w.value} value={w.value}>
                      {w.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <div className="flex items-center justify-between">
              <Button variant="ghost" onClick={() => setStep(1)}>
                <ChevronLeft /> Back
              </Button>
              <Button
                onClick={async () => {
                  if (!collection) {
                    toast.error("Choose a collection");
                    return;
                  }
                  setStep(3);
                }}
              >
                Next <ChevronRight />
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {step === 3 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              {opValue} on {database}.{collection}
            </CardTitle>
            <CardDescription>
              {spec?.hint}. Validation copies the matching documents into a scratch collection, applies the write
              there, and drops it: nothing is changed until this request is approved and applied.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <Field>
              <FieldLabel htmlFor="notes">Notes (why)</FieldLabel>
              <Input
                id="notes"
                placeholder={`${opValue} on ${database}.${collection}`}
                {...register("notes")}
              />
            </Field>

            <Field>
              <FieldLabel>Filter</FieldLabel>
              <JsonEditor
                value={watch("filter")}
                onChange={(v) => setValue("filter", v)}
                height="10rem"
                fields={[]}
                placeholder="{}"
              />
              <FieldError errors={[errors.filter]} />
            </Field>

            <Field>
              <FieldLabel>Value</FieldLabel>
              <JsonEditor
                value={watch("value")}
                onChange={(v) => setValue("value", v)}
                height="10rem"
                fields={[]}
                placeholder={opValue === "delete" ? "—" : '{ "$set": { "field": "value" } }'}
                readOnly={opValue === "delete"}
              />
              <FieldError errors={[errors.value]} />
            </Field>

            <label className="flex items-center gap-2 text-sm">
              <Checkbox checked={watch("many")} onCheckedChange={() => setValue("many", !watch("many"))} />
              many (update/delete every match, insert an array)
            </label>

            <div className="flex flex-wrap items-center gap-2">
              <Button variant="ghost" onClick={() => setStep(2)}>
                <ChevronLeft /> Back
              </Button>
              <span className="flex-1" />
              <span className="flex items-center gap-1.5 text-xs text-amber-500">
                <ShieldAlert className="size-3.5" /> no rollback exists for document changes
              </span>
              <Button onClick={() => void onSend()} disabled={submit.isPending}>
                <Send /> Submit for review
              </Button>
            </div>

            {failure && (
              <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3">
                <H2 className="text-destructive">Validation failed, nothing was recorded</H2>
                <ul className="mt-1.5 grid gap-1 text-xs">
                  {(failure.checks ?? []).map((check) => (
                    <li key={check.target} className={cn(check.ok ? "text-muted-foreground" : "text-destructive")}>
                      {check.ok ? "✓" : "✗"} <span className="font-medium">{check.target}</span>
                      {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Tooltip>
                <TooltipTrigger render={<span />}>
                  <CircleHelp className="size-3.5" />
                </TooltipTrigger>
                <TooltipContent>
                  The dry run proves the filter, the value document and how many documents match. It cannot prove
                  unique-index conflicts, which only appear on apply.
                </TooltipContent>
              </Tooltip>
              What the dry run proves
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
