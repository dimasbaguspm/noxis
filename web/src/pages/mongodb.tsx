import { useEffect, useState } from "react";
import { ChevronRight, Database, Leaf, RefreshCw, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useMongoCollections, useMongoConnections, useMongoDatabases } from "@/hooks/use-mongodb";
import { bytes, compact } from "@/lib/format";
import { cn } from "@/lib/utils";

// MongoExplorePage walks a server: databases with their sizes, then the
// collections of the selected one with count, size and indexes.
export default function MongoExplorePage() {
  const { connections, connection, pick } = useMongoConnections();
  const { databases, loading: loadingDbs, reload: reloadDbs } = useMongoDatabases(connection);
  const [database, setDatabase] = useState<string>();
  const [filter, setFilter] = useState("");

  useEffect(() => {
    if (!database && databases.length > 0) setDatabase(databases[0]?.name);
  }, [database, databases]);

  const { collections, loading: loadingColls, reload: reloadColls } = useMongoCollections(connection, database);

  const needle = filter.trim().toLowerCase();
  const visible = needle
    ? collections.filter((c) => (c.name ?? "").toLowerCase().includes(needle))
    : collections;

  return (
    <div className="flex flex-col gap-4">
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
      </div>

      <div className="grid gap-4 lg:grid-cols-[22rem_1fr]">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Databases</CardTitle>
            <CardDescription>{databases.length} on this server</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-1">
            {databases.length === 0 && (
              <p className="text-sm text-muted-foreground">
                {connection ? "no database visible to this user" : "register a mongodb server to explore"}
              </p>
            )}
            {databases.map((db) => (
              <button
                key={db.name}
                onClick={() => setDatabase(db.name)}
                className={cn(
                  "flex items-center gap-2 rounded-md px-2 py-2 text-left text-sm",
                  db.name === database ? "bg-accent text-accent-foreground" : "hover:bg-accent/50",
                )}
              >
                <Database className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate">{db.name}</span>
                <span className="ml-auto shrink-0 text-xs text-muted-foreground">{bytes(db.sizeBytes ?? 0)}</span>
                <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" />
              </button>
            ))}
          </CardContent>
        </Card>

        <Card className="flex min-h-0 flex-col">
          <CardHeader className="gap-3">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <CardTitle className="text-base">{database ?? "Collections"}</CardTitle>
                <CardDescription>
                  {collections.length} collection(s) · {compact(databases.find((d) => d.name === database)?.objects ?? 0)}{" "}
                  document(s)
                </CardDescription>
              </div>
              <div className="relative max-w-56">
                <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="pl-7"
                  placeholder="filter collections"
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                />
              </div>
            </div>
          </CardHeader>
          <CardContent className="min-h-0 overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Collection</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Documents</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Storage</TableHead>
                  <TableHead>Indexes</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visible.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-muted-foreground">
                      {database ? "no collection matches" : "pick a database"}
                    </TableCell>
                  </TableRow>
                )}
                {visible.map((c) => (
                  <TableRow key={c.name}>
                    <TableCell className="flex items-center gap-2 font-mono text-xs">
                      <Leaf className="size-3.5 text-muted-foreground" />
                      {c.name}
                      {c.capped && (
                        <Badge variant="outline" className="text-[10px]">
                          capped
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="text-[10px]">
                        {c.type}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm tabular-nums">{compact(c.count ?? 0)}</TableCell>
                    <TableCell className="text-sm tabular-nums text-muted-foreground">
                      {bytes(c.sizeBytes ?? 0)}
                    </TableCell>
                    <TableCell className="text-sm tabular-nums text-muted-foreground">
                      {bytes(c.storageBytes ?? 0)}
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
      </div>
    </div>
  );
}
