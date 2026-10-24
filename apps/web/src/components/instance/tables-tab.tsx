import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, Search, Table2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { bytes, compact } from "@/lib/format";
import { PAGE_ROUTES } from "@/lib/routes";

export interface InstanceTable {
  id?: string;
  schema: string;
  name: string;
  comment?: string;
  columns: { name: string; type: string; nullable: boolean }[];
  indexes?: { name: string; columns: string[]; unique?: boolean; primary?: boolean }[];
  estimatedRows?: number;
  sizeBytes?: number;
}

function previewSQL(table: InstanceTable) {
  const name = table.schema ? `${table.schema}.${table.name}` : table.name;
  return `SELECT * FROM ${name} LIMIT 100;`;
}

// TablesTab lists one connection's tables: schema and name filters on the left, the table list on the right.
export function TablesTab({
  tables,
  loading,
  schema,
  onSchema,
  connection,
}: {
  tables: InstanceTable[];
  loading: boolean;
  schema: string;
  onSchema: (schema: string) => void;
  connection: string;
}) {
  const [filter, setFilter] = useState("");
  const schemas = useMemo(() => [...new Set(tables.map((t) => t.schema))].sort(), [tables]);
  const needle = filter.trim().toLowerCase();
  const visible = useMemo(
    () =>
      tables.filter(
        (t) =>
          (!schema || t.schema === schema) &&
          (!needle || `${t.schema}.${t.name}`.toLowerCase().includes(needle)),
      ),
    [tables, schema, needle],
  );

  return (
    <div className="flex h-full min-h-0 flex-col">
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 min-w-0 flex-1">
        <ResizablePanel defaultSize={20} minSize={16}>
          <Card className="mr-2 flex h-full min-h-0 flex-col">
            <CardHeader className="gap-1">
              <CardTitle className="text-base">Tables</CardTitle>
              <CardDescription>Filter by schema or name.</CardDescription>
            </CardHeader>
            <CardContent className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
              <div className="flex flex-col gap-1.5">
                <Label>Schema</Label>
                <Select value={schema || "all"} onValueChange={(v) => onSchema(String(v) === "all" ? "" : String(v))}>
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All schemas</SelectItem>
                    {schemas.map((s) => (
                      <SelectItem key={s} value={s}>
                        {s}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="table-search">Table name</Label>
                <div className="relative">
                  <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    id="table-search"
                    value={filter}
                    onChange={(e) => setFilter(e.target.value)}
                    placeholder="type to filter…"
                    className="pl-7"
                  />
                </div>
              </div>
              <Badge variant="secondary" className="w-fit">
                {visible.length} table(s)
              </Badge>
            </CardContent>
          </Card>
        </ResizablePanel>

        <ResizableHandle withHandle />

        <ResizablePanel defaultSize={80} minSize={40} className="min-w-0">
          <Card className="ml-2 flex h-full min-h-0 flex-col">
            <CardHeader className="gap-1">
              <CardTitle className="text-base">Table list</CardTitle>
              <CardDescription>Rows are planner estimates; size comes from the database catalog.</CardDescription>
            </CardHeader>
            <CardContent className="min-h-0 flex-1 overflow-auto pt-0">
              {loading ? (
                <Skeleton className="h-64 w-full" />
              ) : visible.length === 0 ? (
                <Empty icon={Table2} title="No tables" description="No table matches these filters." />
              ) : (
                <div className="rounded-md border border-border">
                  <Table>
                    <TableHeader className="sticky top-0 bg-card">
                      <TableRow>
                        <TableHead>Table</TableHead>
                        <TableHead>Schema</TableHead>
                        <TableHead>Comment</TableHead>
                        <TableHead>Rows</TableHead>
                        <TableHead>Size</TableHead>
                        <TableHead className="text-right">Records</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {visible.map((t) => (
                        <TableRow key={`${t.schema}.${t.name}`}>
                          <TableCell className="font-mono text-xs">{t.name}</TableCell>
                          <TableCell className="text-xs text-muted-foreground">{t.schema}</TableCell>
                          <TableCell className="max-w-64 truncate text-xs text-muted-foreground" title={t.comment || undefined}>
                            {t.comment || "—"}
                          </TableCell>
                          <TableCell className="text-xs tabular-nums text-muted-foreground">
                            {t.estimatedRows !== undefined ? `~${compact(t.estimatedRows)}` : "—"}
                          </TableCell>
                          <TableCell className="text-xs tabular-nums text-muted-foreground">
                            {t.sizeBytes !== undefined ? bytes(t.sizeBytes) : "—"}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button
                              variant="ghost"
                              size="xs"
                              render={
                                <Link
                                  to={`${PAGE_ROUTES.SQL.QUERY}?connection=${encodeURIComponent(connection)}&sql=${encodeURIComponent(previewSQL(t))}`}
                                />
                              }
                            >
                              <ExternalLink /> Show 100
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
