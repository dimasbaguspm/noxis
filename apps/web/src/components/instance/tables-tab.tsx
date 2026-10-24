import { useMemo } from "react";
import { Table2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Empty } from "@/components/ui/empty";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";

export interface InstanceTable {
  id?: string;
  schema: string;
  name: string;
  columns: { name: string; type: string; nullable: boolean }[];
  indexes?: { name: string; columns: string[]; unique?: boolean; primary?: boolean }[];
  estimatedRows?: number;
  sizeBytes?: number;
}

// TablesTab lists one connection's tables and keeps the schema filter in the URL, so a schema tile can open it directly.
export function TablesTab({
  tables,
  loading,
  schema,
  onSchema,
}: {
  tables: InstanceTable[];
  loading: boolean;
  schema: string;
  onSchema: (schema: string) => void;
}) {
  const schemas = useMemo(() => [...new Set(tables.map((t) => t.schema))].sort(), [tables]);
  const visible = schema ? tables.filter((t) => t.schema === schema) : tables;

  return (
    <div className="flex h-full min-h-0 flex-col gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="grid gap-1.5">
          <Label>Schema</Label>
          <Select value={schema || "all"} onValueChange={(v) => onSchema(String(v) === "all" ? "" : String(v))}>
            <SelectTrigger className="w-56">
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
        <Badge variant="secondary">{visible.length} table(s)</Badge>
      </div>

      {loading ? (
        <Skeleton className="h-64 w-full" />
      ) : visible.length === 0 ? (
        <Empty icon={Table2} title="No tables" description="This schema has no tables to show." />
      ) : (
        <div className="min-h-0 flex-1 overflow-auto rounded-md border border-border">
          <Table>
            <TableHeader className="sticky top-0 bg-card">
              <TableRow>
                <TableHead>Table</TableHead>
                <TableHead>Schema</TableHead>
                <TableHead>Columns</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {visible.map((t) => (
                <TableRow key={`${t.schema}.${t.name}`}>
                  <TableCell className="font-mono text-xs">{t.name}</TableCell>
                  <TableCell className="text-xs text-muted-foreground">{t.schema}</TableCell>
                  <TableCell className="text-xs tabular-nums text-muted-foreground">{t.columns.length}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
