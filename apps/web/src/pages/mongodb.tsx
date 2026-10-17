import { useEffect } from "react";
import { Braces, Database, FileJson, Leaf, ListTree, Play, RefreshCw, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { JsonEditor } from "@/components/sql/json-editor";
import { ResultTable } from "@/components/sql/result-table";
import { JsonTree } from "@/components/mongodb/json-tree";
import { useMongoCollectionsQuery, useMongoConnections, useMongoDatabasesQuery, useMongoRunMutation, useMongoUrlParams } from "@/hooks/use-mongodb";
import { bytes, compact } from "@/lib/format";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";


// MongoExplorePage walks a server and queries it: databases, their
// collections, and the documents of the one you pick. The three panes resize,
// and the documents render as JSON, as a grid or as a tree.
export default function MongoExplorePage() {
  const { connections, connection, pick } = useMongoConnections();
  const { databases, isFetching: loadingDbs, refetch: reloadDbs } = useMongoDatabasesQuery(connection);
  const {
    database: urlDatabase,
    collection,
    filter,
    view,
    setDatabase,
    setCollection,
    setFilter,
    setView,
  } = useMongoUrlParams();
  const database = urlDatabase ?? databases[0]?.name;

  const { collections, isFetching: loadingColls, refetch: reloadColls } = useMongoCollectionsQuery(connection, database);
  const read = useMongoRunMutation();

  // a new collection starts a fresh read
  useEffect(() => {
    read.reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [connection, database, collection]);

  const needle = filter.trim().toLowerCase();
  const visible = needle
    ? collections.filter((c) => (c.name ?? "").toLowerCase().includes(needle))
    : collections;

  const selected = collections.find((c) => c.name === collection);
  const documents = (read.data?.documents ?? []).map((doc) => {
    try {
      return JSON.parse(doc) as unknown;
    } catch {
      return doc;
    }
  });

  function query() {
    if (!connection || !database || !collection) return;
    read.mutate({ connection, body: { op: "find", database, collection, filter, limit: 50 } });
  }

  return (
    <div className="flex h-[calc(100svh-7rem)] flex-col gap-3">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={connection ?? ""} onValueChange={(v) => pick(String(v))}>
          <SelectTrigger className="w-56">
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
        <Button
          variant="outline"
          size="sm"
          disabled={!connection || loadingDbs}
          onClick={() => {
            reloadDbs();
            reloadColls();
          }}
        >
          <RefreshCw className={cn((loadingDbs || loadingColls) && "animate-spin")} /> Refresh
        </Button>
        <span className="text-xs text-muted-foreground">
          reads run here; writes are requested from Requests
        </span>
      </div>

      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={20} minSize={12}>
          <Card className="mr-2 flex h-full min-h-0 flex-col">
            <CardHeader>
              <CardTitle className="text-base">Databases</CardTitle>
              <CardDescription>{databases.length} on this server</CardDescription>
            </CardHeader>
            <CardContent className="min-h-0 overflow-auto">
              <div className="grid gap-1">
                {databases.length === 0 && (
                  <P size="sm">
                    {connection ? "no database visible to this user" : "register a mongodb server to explore"}
                  </P>
                )}
                {databases.map((db) => (
                  <Button
                    key={db.name}
                    variant="ghost"
                    size="sm"
                    className={cn(
                      "h-auto w-full justify-start gap-2 px-2 py-2",
                      db.name === database ? "bg-accent text-accent-foreground hover:bg-accent" : "hover:bg-accent/50",
                    )}
                    onClick={() => setDatabase(db.name)}
                  >
                    <Database className="size-3.5 shrink-0 text-muted-foreground" />
                    <span className="truncate">{db.name}</span>
                    <span className="ml-auto shrink-0 text-xs text-muted-foreground">{bytes(db.sizeBytes ?? 0)}</span>
                  </Button>
                ))}
              </div>
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={38} minSize={18}>
          <Card className="mx-2 flex h-full min-h-0 flex-col">
            <CardHeader className="gap-3">
              <div>
                <CardTitle className="text-base">{database ?? "Collections"}</CardTitle>
                <CardDescription>
                  {collections.length} collection(s) ·{" "}
                  {compact(databases.find((d) => d.name === database)?.objects ?? 0)} document(s)
                </CardDescription>
              </div>
              <div className="relative">
                <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="pl-7"
                  placeholder="filter collections"
                  value={filter.startsWith("{") ? "" : filter}
                  onChange={(e) => setFilter(e.target.value)}
                />
              </div>
            </CardHeader>
            <CardContent className="min-h-0 overflow-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Collection</TableHead>
                    <TableHead>Documents</TableHead>
                    <TableHead>Size</TableHead>
                    <TableHead>Indexes</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} className="text-muted-foreground">
                        {database ? "no collection matches" : "pick a database"}
                      </TableCell>
                    </TableRow>
                  )}
                  {visible.map((c) => (
                    <TableRow
                      key={c.name}
                      className={cn("cursor-pointer", collection === c.name && "bg-accent/60")}
                      onClick={() => setCollection(c.name)}
                    >
                      <TableCell className="flex items-center gap-2 font-mono text-xs">
                        <Leaf className="size-3.5 text-muted-foreground" />
                        {c.name}
                        {c.capped && (
                          <Badge variant="outline" className="text-[10px]">
                            capped
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-sm tabular-nums">{compact(c.count ?? 0)}</TableCell>
                      <TableCell className="text-xs tabular-nums text-muted-foreground">
                        {bytes(c.sizeBytes ?? 0)}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {c.indexes ?? 0} · {bytes(c.indexSizeBytes ?? 0)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={42} minSize={20}>
          <Card className="ml-2 flex h-full min-h-0 flex-col">
            <CardHeader className="gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <CardTitle className="font-mono text-base">
                  {collection ? `${database}.${collection}` : "Documents"}
                </CardTitle>
                {selected && (
                  <Badge variant="outline">
                    {compact(selected.count ?? 0)} docs · {bytes(selected.sizeBytes ?? 0)}
                  </Badge>
                )}
                <span className="flex-1" />
                <div className="flex gap-0.5">
                  {(
                    [
                      { id: "json", label: "JSON", icon: <FileJson className="size-3.5" /> },
                      { id: "raw", label: "Raw", icon: <Braces className="size-3.5" /> },
                      { id: "tree", label: "Tree", icon: <ListTree className="size-3.5" /> },
                    ] as const
                  ).map((mode) => (
                    <Button
                      key={mode.id}
                      size="xs"
                      variant={view === mode.id ? "secondary" : "ghost"}
                      onClick={() => setView(mode.id)}
                    >
                      {mode.icon} {mode.label}
                    </Button>
                  ))}
                </div>
              </div>
              <CardDescription>
                {collection
                  ? "filter the documents, then read them as JSON, as a grid or as a tree"
                  : "pick a collection to query it"}
              </CardDescription>
            </CardHeader>
            <CardContent className="flex min-h-0 flex-1 flex-col gap-3">
              <div className="flex items-start gap-2">
                <div className="min-w-0 flex-1">
                  <JsonEditor
                    value={filter.startsWith("{") ? filter : "{}"}
                    onChange={setFilter}
                    height="6rem"
                    fields={read.data?.columns ?? []}
                    placeholder="{}"
                  />
                </div>
                <Button onClick={query} disabled={read.isPending || !collection} className="mt-1">
                  <Play /> Run
                </Button>
              </div>

              <div className="min-h-0 flex-1 overflow-auto rounded-md border border-border">
                {!read.data ? (
                  <P size="sm" className="p-3">no documents loaded yet</P>
                ) : view === "raw" ? (
                  <ResultTable result={read.data} title={`${database}.${collection}`} />
                ) : view === "json" ? (
                  <pre className="p-3 font-mono text-xs whitespace-pre-wrap">
                    {JSON.stringify(documents, null, 2)}
                  </pre>
                ) : (
                  <div className="p-3">
                    <JsonTree value={documents} />
                  </div>
                )}
              </div>

              {read.data && (
                <P>
                  {read.data.message} · {read.data.durationMs ?? 0} ms
                </P>
              )}
            </CardContent>
          </Card>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
