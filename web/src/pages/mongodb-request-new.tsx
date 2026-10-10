import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ChevronLeft, ChevronRight, CircleHelp, Send, ShieldAlert } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { JsonEditor } from "@/components/sql/json-editor";
import { useMongoCollectionsQuery, useMongoConnections, useMongoDatabasesQuery, useMongoSubmitChangeMutation } from "@/hooks/use-mongodb";
import type { Change } from "@/lib/api";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

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

// MongoRequestNewPage is the request wizard: pick the database, pick the
// collection, write the filter and the value, submit for review. Nothing is
// applied here — validation runs on a scratch collection and the request waits
// in the queue.
export default function MongoRequestNewPage() {
  const navigate = useNavigate();
  const { connections, connection, pick } = useMongoConnections();
  const { databases } = useMongoDatabasesQuery(connection);
  const [step, setStep] = useState(1);
  const [database, setDatabase] = useState<string>();
  const [collection, setCollection] = useState<string>();
  const [op, setOp] = useState<string>("update");
  const [notes, setNotes] = useState("");
  const [filter, setFilter] = useState(SAMPLES.update?.filter ?? "");
  const [value, setValue] = useState(SAMPLES.update?.value ?? "");
  const [many, setMany] = useState(false);
  const [failure, setFailure] = useState<Change>();

  const { collections } = useMongoCollectionsQuery(connection, database);
  const submit = useMongoSubmitChangeMutation();

  useEffect(() => {
    if (!database && databases.length > 0) setDatabase(databases[0]?.name);
  }, [database, databases]);
  useEffect(() => {
    if (collection && collections.some((c) => c.name === collection)) return;
    setCollection(collections[0]?.name);
  }, [collection, collections]);

  const spec = WRITES.find((w) => w.value === op);

  function chooseOp(next: string) {
    setOp(next);
    setFilter(SAMPLES[next]?.filter ?? "");
    setValue(SAMPLES[next]?.value ?? "");
    setMany(false);
  }

  async function send() {
    if (!connection || !database || !collection) return;
    setFailure(undefined);
    try {
      const change = (await submit.mutateAsync({
        connection,
        notes: notes.trim() || `${op} on ${database}.${collection}`,
        op,
        database,
        collection,
        filter,
        value,
        many,
      })) as Change;
      if (change.status === "failed") {
        // a failed validation is a rejected proposal: it is not in the queue
        setFailure(change);
        toast.error("validation failed — the request was not recorded");
        return;
      }
      toast.success(`request ${change.status}: ${change.estAffected ?? 0} document(s) would be affected`);
      navigate(PAGE_ROUTES.MONGODB.REQUEST_DETAIL(change.id ?? ""));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  }

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
            <CardDescription>the request is validated against this database</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Field>
              <FieldLabel>Connection</FieldLabel>
              <Select value={connection ?? ""} onValueChange={(v) => pick(String(v))}>
                <SelectTrigger>
                  <SelectValue placeholder="no mongodb registered" />
                </SelectTrigger>
                <SelectContent>
                  {connections.map((c) => (
                    <SelectItem key={c.name} value={c.name}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>Database</FieldLabel>
              <Select value={database ?? ""} onValueChange={(v) => setDatabase(String(v))}>
                <SelectTrigger>
                  <SelectValue placeholder="pick a database" />
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
              <Button disabled={!connection || !database} onClick={() => setStep(2)}>
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
              <Select value={collection ?? ""} onValueChange={(v) => setCollection(String(v))}>
                <SelectTrigger>
                  <SelectValue placeholder="pick a collection" />
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
              <Select value={op} onValueChange={(v) => chooseOp(String(v))}>
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
              <Button disabled={!collection} onClick={() => setStep(3)}>
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
              {op} on {database}.{collection}
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
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder={`${op} on ${database}.${collection}`}
              />
            </Field>

            <Field>
              <FieldLabel>Filter</FieldLabel>
              <JsonEditor
                value={filter}
                onChange={setFilter}
                height="10rem"
                fields={[]}
                placeholder="{}"
              />
            </Field>

            <Field>
              <FieldLabel>Value</FieldLabel>
              <JsonEditor
                value={value}
                onChange={setValue}
                height="10rem"
                fields={[]}
                placeholder={op === "delete" ? "—" : '{ "$set": { "field": "value" } }'}
                readOnly={op === "delete"}
              />
            </Field>

            <label className="flex items-center gap-2 text-sm">
              <Checkbox checked={many} onCheckedChange={() => setMany((v) => !v)} />
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
              <Button onClick={() => void send()} disabled={submit.isPending}>
                <Send /> Submit for review
              </Button>
            </div>

            {failure && (
              <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3">
                <p className="text-sm font-medium text-destructive">validation failed, nothing was recorded</p>
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
              what the dry run proves
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
