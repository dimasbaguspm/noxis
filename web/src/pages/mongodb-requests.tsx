import { useEffect, useMemo, useState } from "react";
import { CircleHelp, Play, ShieldAlert } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { JsonEditor } from "@/components/sql/json-editor";
import { ResultTable } from "@/components/sql/result-table";
import { useMongoCollections, useMongoConnections, useMongoDatabases, useMongoRequest } from "@/hooks/use-mongodb";
import type { MongoRequest } from "@/lib/api";
import { cn } from "@/lib/utils";

// Reads run straight away; writes are the ops that change documents, and the
// backend refuses an update or delete without a filter.
const OPS = [
  { value: "find", label: "find", write: false, hint: "filter, optional sort and limit" },
  { value: "aggregate", label: "aggregate", write: false, hint: "a pipeline array in the filter box" },
  { value: "count", label: "count", write: false, hint: "filter only" },
  { value: "insert", label: "insert", write: true, hint: "a document, or an array with many" },
  { value: "update", label: "update", write: true, hint: "filter plus an update document ($set, $inc…)" },
  { value: "delete", label: "delete", write: true, hint: "filter is required" },
] as const;

const SAMPLES: Record<string, { filter: string; value: string }> = {
  find: { filter: '{ "status": "active" }', value: "" },
  aggregate: { filter: '[{ "$match": {} }, { "$count": "n" }]', value: "" },
  count: { filter: "{}", value: "" },
  insert: { filter: "", value: '{ "name": "example", "qty": 1 }' },
  update: { filter: '{ "_id": { "$oid": "…" } }', value: '{ "$set": { "qty": 2 } }' },
  delete: { filter: '{ "name": "example" }', value: "" },
};

// MongoRequestsPage is the document console: one request at a time, with the
// filter and the value kept apart so each gets its own editor and its own
// suggestions.
export default function MongoRequestsPage() {
  const { connections, connection, pick } = useMongoConnections();
  const { databases } = useMongoDatabases(connection);
  const [database, setDatabase] = useState<string>();
  const [collection, setCollection] = useState<string>();
  const [op, setOp] = useState<string>("find");
  const [filter, setFilter] = useState(SAMPLES.find?.filter ?? "{}");
  const [value, setValue] = useState("");
  const [sort, setSort] = useState("");
  const [limit, setLimit] = useState("50");
  const [many, setMany] = useState(false);

  useEffect(() => {
    if (!database && databases.length > 0) setDatabase(databases[0]?.name);
  }, [database, databases]);

  const { collections } = useMongoCollections(connection, database);
  useEffect(() => {
    if (collection && collections.some((c) => c.name === collection)) return;
    setCollection(collections[0]?.name);
  }, [collection, collections]);

  const { result, run, loading } = useMongoRequest();

  // the fields the completion suggests come from the collection's own documents
  const fields = useMemo(() => {
    const names = new Set<string>(["_id"]);
    for (const column of result?.columns ?? []) names.add(column);
    return [...names];
  }, [result]);

  const spec = OPS.find((o) => o.value === op);
  const isAggregate = op === "aggregate";

  function choose(next: string) {
    setOp(next);
    const sample = SAMPLES[next];
    setFilter(sample?.filter ?? "");
    setValue(sample?.value ?? "");
    setMany(false);
  }

  function submit() {
    if (!connection || !database || !collection) return;
    const body: MongoRequest = {
      op,
      database,
      collection,
      filter,
      value,
      sort,
      limit: Number(limit) || 0,
      many,
    };
    void run.run(connection, body);
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end gap-3">
        <Select value={connection ?? ""} onValueChange={(v) => pick(String(v))}>
          <SelectTrigger className="w-52">
            <SelectValue placeholder="no mongodb registered" />
          </SelectTrigger>
          <SelectContent>
            {connections.length === 0 && <SelectItem value="none">none registered yet</SelectItem>}
            {connections.map((c) => (
              <SelectItem key={c.name} value={c.name}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={database ?? ""} onValueChange={(v) => setDatabase(String(v))}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="database" />
          </SelectTrigger>
          <SelectContent>
            {databases.map((db) => (
              <SelectItem key={db.name} value={db.name ?? ""}>
                {db.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={collection ?? ""} onValueChange={(v) => setCollection(String(v))}>
          <SelectTrigger className="w-52">
            <SelectValue placeholder="collection" />
          </SelectTrigger>
          <SelectContent>
            {collections.map((c) => (
              <SelectItem key={c.name} value={c.name ?? ""}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={op} onValueChange={(v) => choose(String(v))}>
          <SelectTrigger className="w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {OPS.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Badge variant={spec?.write ? "destructive" : "secondary"}>{spec?.write ? "write" : "read"}</Badge>
        <Tooltip>
          <TooltipTrigger render={<span className="text-muted-foreground" />}>
            <CircleHelp className="size-3.5" />
          </TooltipTrigger>
          <TooltipContent>
            {spec?.write
              ? "Runs immediately. There is no rollback for documents, so update and delete need a filter."
              : "Reads run immediately."}
          </TooltipContent>
        </Tooltip>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{isAggregate ? "Pipeline" : "Filter"}</CardTitle>
            <CardDescription>{spec?.hint}</CardDescription>
          </CardHeader>
          <CardContent>
            <JsonEditor value={filter} onChange={setFilter} fields={fields} height="12rem" placeholder="{}" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Value</CardTitle>
            <CardDescription>
              {spec?.write ? "the document to write" : "not used by this op"}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <JsonEditor
              value={value}
              onChange={setValue}
              fields={fields}
              height="12rem"
              placeholder={spec?.write ? '{ "$set": { "field": "value" } }' : "—"}
              readOnly={!spec?.write}
            />
          </CardContent>
        </Card>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="grid gap-1.5">
          <Label htmlFor="mongo-limit">Limit</Label>
          <Input
            id="mongo-limit"
            className="w-24"
            value={limit}
            onChange={(e) => setLimit(e.target.value)}
            disabled={op !== "find"}
          />
        </div>
        <div className="grid gap-1.5">
          <Label htmlFor="mongo-sort">Sort (JSON)</Label>
          <Input
            id="mongo-sort"
            className="w-56"
            placeholder='{ "qty": -1 }'
            value={sort}
            onChange={(e) => setSort(e.target.value)}
            disabled={op !== "find"}
          />
        </div>
        <label className="flex items-center gap-2 pb-2 text-sm">
          <input type="checkbox" checked={many} disabled={!spec?.write} onChange={() => setMany((v) => !v)} />
          many
        </label>
        <Button onClick={submit} disabled={loading || !connection || !collection}>
          <Play /> Run request
        </Button>
        {spec?.write && (
          <span className="flex items-center gap-1.5 pb-2 text-xs text-amber-500">
            <ShieldAlert className="size-3.5" /> writes are immediate
          </span>
        )}
      </div>

      <Card className={cn("min-h-0 flex-1", !result && "opacity-70")}>
        <CardHeader>
          <CardTitle className="text-base">Result</CardTitle>
          <CardDescription>
            {result?.message ? `${result.message} · ${result.durationMs ?? 0} ms` : "run a request to see documents"}
          </CardDescription>
        </CardHeader>
        <CardContent className="min-h-0 overflow-auto">
          <ResultTable result={result} title={`${database ?? ""}.${collection ?? ""}`} />
        </CardContent>
      </Card>
    </div>
  );
}
