import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, Play } from "lucide-react";
import { JsonTree } from "@/components/mongodb/json-tree";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  useMongoCollectionsQuery,
  useMongoConnections,
  useMongoDatabasesQuery,
  useMongoRunMutation,
} from "@/hooks/use-mongodb";
import { PAGE_ROUTES } from "@/lib/routes";
import { P } from "@/components/ui/typography";

// MongoExplorer is a compact find console on the instance page: database, collection, filter and the first 50 documents.
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
    <Card>
      <CardHeader className="gap-1">
        <CardTitle className="text-base">Explore</CardTitle>
        <CardDescription>
          Read the first 50 documents of a collection. Writes go through the request flow on the full explorer.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Select value={db ?? ""} onValueChange={(v) => setDatabase(String(v))}>
            <SelectTrigger className="w-44" aria-label="database">
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
            <SelectTrigger className="w-52" aria-label="collection">
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
            className="h-8 min-w-40 flex-1 font-mono text-xs"
            placeholder="{}"
            aria-label="filter"
          />
          <Button size="sm" onClick={query} disabled={!db || !coll || run.isPending}>
            <Play /> Run
          </Button>
          <Button
            size="sm"
            variant="outline"
            render={
              <Link to={PAGE_ROUTES.MONGODB.EXPLORE} onClick={() => pick(connection)} />
            }
          >
            <ExternalLink /> Full explorer
          </Button>
        </div>

        {run.error instanceof Error && <p className="text-xs text-destructive">{run.error.message}</p>}

        <div className="h-72 overflow-auto rounded-md border border-border p-2">
          {!result ? (
            <P size="sm">No documents loaded yet.</P>
          ) : documents.length === 0 ? (
            <P size="sm">No documents match this filter.</P>
          ) : (
            <div className="flex flex-col gap-2">
              {documents.map((doc, i) => (
                <div key={i} className="border-b border-border pb-2 last:border-0 last:pb-0">
                  <JsonTree value={doc} />
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
