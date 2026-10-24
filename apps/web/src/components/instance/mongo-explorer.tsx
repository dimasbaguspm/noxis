import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, FileJson, Play } from "lucide-react";
import { JsonTree } from "@/components/mongodb/json-tree";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Workspace } from "@/components/layout/workspace";
import {
  useMongoCollectionsQuery,
  useMongoConnections,
  useMongoDatabasesQuery,
  useMongoRunMutation,
} from "@/hooks/use-mongodb";
import { PAGE_ROUTES } from "@/lib/routes";

// MongoExplorer is a compact find console on the instance page: filters on the left, the first 50 documents on the right.
export function MongoExplorer({ connection }: { connection: string }) {
  const databases = useMongoDatabasesQuery(connection);
  const [database, setDatabase] = useState("");
  const db = databases.databases.some((d) => d.name === database)
    ? database
    : databases.databases[0]?.name;

  const collections = useMongoCollectionsQuery(connection, db);
  const [collection, setCollection] = useState("");
  const coll = collections.collections.some((c) => c.name === collection)
    ? collection
    : collections.collections[0]?.name;

  const [filter, setFilter] = useState("{}");
  const run = useMongoRunMutation();
  const { pick } = useMongoConnections();

  useEffect(() => {
    setFilter("{}");
  }, [db, coll]);

  const target = run.variables;
  const matches =
    !!target &&
    target.connection === connection &&
    target.body.database === db &&
    target.body.collection === coll &&
    target.body.filter === filter;
  const result = matches ? run.data : undefined;
  const documents = (result?.documents ?? []).map((doc) => {
    try {
      return JSON.parse(doc) as unknown;
    } catch {
      return doc;
    }
  });

  function query() {
    if (!db || !coll || run.isPending) return;
    run.mutate({ connection, body: { op: "find", database: db, collection: coll, filter, limit: 50 } });
  }

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-3">
            <Select value={db ?? ""} onValueChange={(v) => setDatabase(String(v))}>
              <SelectTrigger className="w-full" aria-label="database">
                <SelectValue placeholder={databases.isFetching ? "loading…" : "database"} />
              </SelectTrigger>
              <SelectContent>
                {databases.databases.map((d) => (
                  <SelectItem key={d.name} value={d.name ?? ""}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={coll ?? ""} onValueChange={(v) => setCollection(String(v))}>
              <SelectTrigger className="w-full" aria-label="collection">
                <SelectValue placeholder={collections.isFetching ? "loading…" : "collection"} />
              </SelectTrigger>
              <SelectContent>
                {collections.collections.map((c) => (
                  <SelectItem key={c.name} value={c.name ?? ""}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="h-8 font-mono text-xs"
              placeholder="{}"
              aria-label="filter"
            />
            <div className="flex flex-wrap gap-2">
              <Button size="sm" onClick={query} disabled={!db || !coll || run.isPending}>
                <Play /> Run
              </Button>
              <Button
                size="sm"
                variant="outline"
                render={<Link to={PAGE_ROUTES.MONGODB.EXPLORE} onClick={() => pick(connection)} />}
              >
                <ExternalLink /> Full explorer
              </Button>
            </div>
            {run.error instanceof Error && <p className="text-xs text-destructive">{run.error.message}</p>}
          </div>
        }
        right={
          !result ? (
            <Empty icon={FileJson} title="No documents" description="Run the filter to load the first 50 documents." />
          ) : documents.length === 0 ? (
            <Empty icon={FileJson} title="No matches" description="No document matches this filter." />
          ) : (
            <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-auto">
              {documents.map((doc, i) => (
                <div key={i} className="border-b border-border pb-2 last:border-0 last:pb-0">
                  <JsonTree value={doc} />
                </div>
              ))}
            </div>
          )
        }
      />
    </div>
  );
}
