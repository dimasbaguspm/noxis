import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, Search, Table2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { Workspace } from "@/components/layout/workspace";
import { bytes, compact } from "@/lib/format";
import { PAGE_ROUTES } from "@/constants/routes";

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
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-col gap-1.5">
              <Label>Schema</Label>
              <Select
                value={schema || "all"}
                onValueChange={(v) => onSchema(String(v) === "all" ? "" : String(v))}
              >
                <SelectTrigger className="w-full">
                  <SelectValue>
                    {(value) => (value === "all" ? "All schemas" : String(value))}
                  </SelectValue>
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
              <Label>Search tables</Label>
              <div className="relative">
                <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="table-search"
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                  placeholder="Search tables"
                  className="pl-7"
                />
              </div>
            </div>
          </div>
        }
        right={
          loading ? (
            <Skeleton className="h-64 w-full" />
          ) : visible.length === 0 ? (
            <Empty icon={Table2} title="No tables" description="No table matches these filters." />
          ) : (
            <div className="min-h-0 flex-1 overflow-auto">
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
                        <TableCell
                          className="max-w-64 truncate text-xs text-muted-foreground"
                          title={t.comment || undefined}
                        >
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
                                to={`${PAGE_ROUTES.QUERY}?connection=${encodeURIComponent(connection)}&sql=${encodeURIComponent(previewSQL(t))}`}
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
            </div>
          )
        }
      />
    </div>
  );
}
