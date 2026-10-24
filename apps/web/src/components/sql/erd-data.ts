import type { SchemaGraph } from "@/lib/api";
import type { ErdTable } from "@/components/sql/erd-canvas";

// toErdTables normalises the generated optional fields once, so the page and the instance tab render the same graph.
export function toErdTables(data?: SchemaGraph): ErdTable[] {
  return (data?.tables ?? []).map((table) => ({
    id: table.id ?? `${table.schema ?? ""}.${table.name ?? ""}`,
    schema: table.schema ?? "",
    name: table.name ?? "",
    columns: (table.columns ?? []).map((col) => ({
      name: col.name ?? "",
      type: col.type ?? "",
      nullable: col.nullable ?? false,
      primaryKey: col.primaryKey,
      unique: col.unique,
      references: col.references,
    })),
    indexes: (table.indexes ?? []).map((idx) => ({
      name: idx.name ?? "",
      columns: idx.columns ?? [],
      unique: idx.unique,
      primary: idx.primary,
    })),
    estimatedRows: table.estimatedRows,
  }));
}
