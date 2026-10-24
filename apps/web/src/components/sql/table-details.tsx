import { Link } from "react-router-dom";
import { Network } from "lucide-react";
import type { ErdTable } from "@/components/sql/erd-canvas";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { compact } from "@/lib/format";
import { PAGE_ROUTES } from "@/lib/routes";

export function TableDetails({ table, tables, connection }: { table: ErdTable; tables: ErdTable[]; connection?: string }) {
  const incoming = tables.flatMap((t) =>
    t.columns
      .filter((col) => col.references?.table === table.name)
      .map((col) => ({ from: t.name, column: col.name, to: col.references?.column ?? "" })),
  );

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap gap-1.5">
        <Badge variant="secondary">{table.columns.length} columns</Badge>
        {table.estimatedRows !== undefined && <Badge variant="secondary">~{compact(table.estimatedRows)} rows</Badge>}
        <Badge variant="secondary">{table.indexes?.length ?? 0} indexes</Badge>
      </div>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">Columns</h3>
        <div className="grid gap-1">
          {table.columns.map((col) => (
            <div key={col.name} className="flex items-center gap-2 rounded-md border px-2 py-1.5 text-xs">
              <span className="w-5 shrink-0 font-mono text-[10px] text-muted-foreground">
                {col.primaryKey ? "PK" : col.references ? "FK" : col.unique ? "UK" : ""}
              </span>
              <span className="truncate font-mono">{col.name}</span>
              <span className="ml-auto truncate text-muted-foreground">{col.type}</span>
              {col.nullable && <Badge variant="outline">NULL</Badge>}
            </div>
          ))}
        </div>
      </section>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">Indexes</h3>
        {table.indexes && table.indexes.length > 0 ? (
          <div className="grid gap-1">
            {table.indexes.map((idx) => (
              <div key={idx.name} className="rounded-md border px-2 py-1.5 text-xs">
                <div className="flex items-center gap-1.5">
                  <span className="truncate font-mono">{idx.name}</span>
                  {idx.primary && <Badge variant="outline">primary</Badge>}
                  {idx.unique && !idx.primary && <Badge variant="outline">unique</Badge>}
                </div>
                <div className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">
                  {idx.columns.length > 0 ? idx.columns.join(", ") : "expression"}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No indexes reported.</p>
        )}
      </section>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">References</h3>
        {table.columns.some((col) => col.references) ? (
          <div className="grid gap-1">
            {table.columns
              .filter((col) => col.references)
              .map((col) => (
                <div key={col.name} className="rounded-md border px-2 py-1.5 font-mono text-xs">
                  {col.name} → {col.references?.schema}.{col.references?.table}.{col.references?.column}
                </div>
              ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No declared foreign keys.</p>
        )}
      </section>

      <section>
        <h3 className="mb-2 text-xs font-medium text-muted-foreground uppercase">Incoming references</h3>
        {incoming.length > 0 ? (
          <div className="grid gap-1">
            {incoming.map((ref) => (
              <div key={`${ref.from}.${ref.column}`} className="rounded-md border px-2 py-1.5 font-mono text-xs">
                {ref.from}.{ref.column} → {table.name}.{ref.to}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No table references this one.</p>
        )}
      </section>

      <Button variant="outline" size="sm" render={<Link to={`${PAGE_ROUTES.SQL.QUERY}?connection=${encodeURIComponent(connection ?? "")}`} />}>
        <Network /> Open SQL query
      </Button>
    </div>
  );
}
