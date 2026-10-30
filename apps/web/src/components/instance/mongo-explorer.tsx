import { useMongoExplorer } from "@/hooks/use-mongo-explorer";
import { Link } from "react-router-dom";
import { ExternalLink, FileJson, Play } from "lucide-react";
import { JsonTree } from "@/components/mongodb/json-tree";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Workspace } from "@/components/layout/workspace";
import { useMongoConnections } from "@/hooks/use-mongodb";
import { PAGE_ROUTES } from "@/constants/routes";

// MongoExplorer is a compact find console on the instance page: filters on the left, the first 50 documents on the right.
export function MongoExplorer({ connection }: { connection: string }) {
  const {
    databases,
    db,
    setDatabase,
    collections,
    coll,
    setCollection,
    filter,
    setFilter,
    run,
    result,
    documents,
    query,
  } = useMongoExplorer(connection);
  const { pick } = useMongoConnections();

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-3">
            <Select value={db ?? ""} onValueChange={(v) => setDatabase(String(v))}>
              <SelectTrigger className="w-full" aria-label="Database">
                <SelectValue placeholder={databases.isFetching ? "Loading…" : "Database"} />
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
              <SelectTrigger className="w-full" aria-label="Collection">
                <SelectValue placeholder={collections.isFetching ? "Loading…" : "Collection"} />
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
              aria-label="Filter"
            />
            <div className="flex flex-wrap gap-2">
              <Button size="sm" onClick={query} disabled={!db || !coll || run.isPending}>
                <Play /> Run
              </Button>
              <Button
                size="sm"
                variant="outline"
                render={<Link to={PAGE_ROUTES.EXPLORE} onClick={() => pick(connection)} />}
              >
                <ExternalLink /> Full explorer
              </Button>
            </div>
            {run.error instanceof Error && (
              <p className="text-xs text-destructive">{run.error.message}</p>
            )}
          </div>
        }
        right={
          !result ? (
            <Empty
              icon={FileJson}
              title="No documents"
              description="Run the filter to load the first 50 documents."
            />
          ) : documents.length === 0 ? (
            <Empty
              icon={FileJson}
              title="No matches"
              description="No document matches this filter."
            />
          ) : (
            <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-auto">
              {documents.map((doc) => (
                <div
                  key={
                    typeof doc === "object" && doc !== null && "_id" in doc
                      ? String((doc as { _id: unknown })._id)
                      : JSON.stringify(doc)
                  }
                  className="border-b border-border pb-2 last:border-0 last:pb-0"
                >
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
