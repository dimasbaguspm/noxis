import { useMemo, useState } from "react";
import { debounce, group } from "radash";
import type { Table } from "@/lib/api";

// useTableBrowser turns the flat table list into schema groups, a debounced search and one expanded table.
export function useTableBrowser(tables: Table[]) {
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [selectedSchema, setSelectedSchema] = useState("");

  const setFilterDebounced = useMemo(
    () => debounce({ delay: 200 }, (v: string) => setQuery(v)),
    [],
  );

  // search shows the keystroke immediately and filters the list once typing pauses.
  function search(value: string) {
    setFilter(value);
    void setFilterDebounced(value);
  }

  const schemas = useMemo(
    () => Object.keys(group(tables, (t) => t.schema || "public")).sort(),
    [tables],
  );
  const schemaFilter = schemas.includes(selectedSchema) ? selectedSchema : "";

  const entries = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const scope = schemaFilter
      ? tables.filter((t) => (t.schema || "public") === schemaFilter)
      : tables;
    const matching = needle
      ? scope.filter((t) => `${t.schema}.${t.name}`.toLowerCase().includes(needle))
      : scope;
    const groups = group(matching, (t) => t.schema || "public");
    return Object.entries(groups) as [string, Table[]][];
  }, [tables, query, schemaFilter]);

  // selecting "all" clears the schema filter; the empty string means no filter.
  function selectSchema(value: string) {
    setSelectedSchema(value === "all" ? "" : value);
  }

  // a second click on the open table collapses it.
  function expand(key: string) {
    setExpanded((current) => (current === key ? null : key));
  }

  return { filter, search, expanded, expand, selectSchema, schemas, schemaFilter, entries };
}
