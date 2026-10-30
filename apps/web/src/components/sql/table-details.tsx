import { Link } from "react-router-dom";
import { Network } from "lucide-react";
import type { ErdTable } from "@/components/sql/erd-canvas";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { bytes, compact } from "@/lib/format";
import { PAGE_ROUTES } from "@/constants/routes";

function previewSQL(table: ErdTable) {
  const name = table.schema ? `${table.schema}.${table.name}` : table.name;
  return `SELECT * FROM ${name} LIMIT 100;`;
}

// TableDetails is the ERD sheet: compact metrics first, then columns, indexes and declared references.
export function TableDetails({
  table,
  tables,
  connection,
  instanceId,
}: {
  table: ErdTable;
  tables: ErdTable[];
  connection?: string;
  instanceId?: string;
}) {
  const incoming = tables.flatMap((t) =>
    t.columns
      .filter(
        (col) =>
          col.references?.table === table.name &&
          (!col.references?.schema || !table.schema || col.references.schema === table.schema),
      )
      .map((col) => ({ from: t.name, column: col.name, to: col.references?.column ?? "" })),
  );
  const outgoing = table.columns.filter((col) => col.references);
  const target = instanceId
    ? `instance=${encodeURIComponent(instanceId)}`
    : connection
      ? `connection=${encodeURIComponent(connection)}`
      : "";

  return (
    <div className="flex flex-col gap-5">
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Metric label="Columns" value={String(table.columns.length)} />
        <Metric
          label="Rows"
          value={table.estimatedRows !== undefined ? `~${compact(table.estimatedRows)}` : "—"}
        />
        <Metric label="Size" value={table.sizeBytes !== undefined ? bytes(table.sizeBytes) : "—"} />
        <Metric label="Indexes" value={String(table.indexes?.length ?? 0)} />
      </div>

      <section>
        <h3 className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Columns
        </h3>
        <div className="overflow-hidden rounded-md border">
          {table.columns.map((col) => (
            <div
              key={col.name}
              className="flex items-center gap-2 border-b px-2 py-1.5 text-xs last:border-0"
            >
              <span className="w-6 shrink-0 font-mono text-[10px] text-muted-foreground">
                {col.primaryKey ? "PK" : col.references ? "FK" : col.unique ? "UK" : ""}
              </span>
              <span className="min-w-0 flex-1 truncate font-mono">{col.name}</span>
              <span className="max-w-[45%] truncate text-muted-foreground">{col.type}</span>
              {col.nullable && <Badge variant="outline">NULL</Badge>}
            </div>
          ))}
        </div>
      </section>

      <div className="grid gap-4 sm:grid-cols-2">
        <section>
          <h3 className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Indexes
          </h3>
          {table.indexes && table.indexes.length > 0 ? (
            <div className="grid gap-1">
              {table.indexes.map((idx) => (
                <div key={idx.name} className="rounded-md border px-2 py-1.5 text-xs">
                  <div className="flex items-center gap-1.5">
                    <span className="truncate font-mono">{idx.name}</span>
                    {idx.primary && <Badge variant="outline">Primary</Badge>}
                    {idx.unique && !idx.primary && <Badge variant="outline">Unique</Badge>}
                  </div>
                  <div className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">
                    {idx.columns.length > 0 ? idx.columns.join(", ") : "Expression"}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">No indexes reported.</p>
          )}
        </section>

        <section>
          <h3 className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            References
          </h3>
          {outgoing.length > 0 ? (
            <div className="grid gap-1">
              {outgoing.map((col) => (
                <div
                  key={col.name}
                  className="truncate rounded-md border px-2 py-1.5 font-mono text-xs"
                >
                  {col.name} → {col.references?.schema}.{col.references?.table}.
                  {col.references?.column}
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">No declared foreign keys.</p>
          )}
          {incoming.length > 0 && (
            <div className="mt-2 grid gap-1">
              {incoming.map((ref) => (
                <div
                  key={`${ref.from}.${ref.column}`}
                  className="truncate rounded-md border px-2 py-1.5 font-mono text-xs"
                >
                  {ref.from}.{ref.column} → {table.name}.{ref.to}
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      <Button
        variant="outline"
        size="sm"
        render={
          <Link
            to={`${PAGE_ROUTES.QUERY}?${target}${target ? "&" : ""}sql=${encodeURIComponent(previewSQL(table))}`}
          />
        }
      >
        <Network /> Open SQL query
      </Button>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border px-2 py-1.5">
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className="truncate text-sm font-medium tabular-nums">{value}</div>
    </div>
  );
}
