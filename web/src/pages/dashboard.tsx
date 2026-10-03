import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { debounce, group } from "radash";
import { Search } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { ConnectionList } from "@/components/connection/connection-list";
import { useApp } from "@/app/app-provider";
import { useChanges } from "@/hooks/use-data";

export default function DashboardPage() {
  const { connections, schema, active } = useApp();
  const { total, changes } = useChanges();

  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);

  const found = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return [];
    return schema.filter((t) => `${t.schema}.${t.name}`.toLowerCase().includes(needle)).slice(0, 20);
  }, [schema, query]);

  const byStatus = (statuses: string[]) => changes.filter((c) => statuses.includes(c.status ?? "")).length;
  const applied = byStatus(["applied"]);
  const pending = byStatus(["validated", "approved"]);
  const failed = byStatus(["failed"]);
  const schemaCount = Object.keys(group(schema, (t) => t.schema || "public")).length;

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Stat title="Connections" value={connections.length} hint={active ? `active: ${active}` : "none active"} />
        <Stat
          title="Tables"
          value={schema.length}
          hint={active ? `${schemaCount} schema(s) in ${active}` : "pick a connection"}
        />
        <Stat title="Pending changes" value={pending} hint="validated or approved" />
        <Stat title="Applied / failed" value={`${applied} / ${failed}`} hint={`${total} changes total`} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Find a table</CardTitle>
          <CardDescription>search across {active ?? "the active connection"}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="relative max-w-md">
            <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-7"
              placeholder="table name"
              value={filter}
              onChange={(e) => {
                setFilter(e.target.value);
                void setFilterDebounced(e.target.value);
              }}
            />
          </div>
          {found.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {found.map((t) => (
                <Link
                  key={`${t.schema}.${t.name}`}
                  to="/query"
                  className="rounded-full border border-border px-2.5 py-1 text-xs hover:border-primary/50"
                >
                  {t.schema}.{t.name}
                  <span className="ml-1 text-muted-foreground">{t.columns.length}</span>
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">Connections</h2>
        <span className="flex-1" />
        <Link to="/query" className="text-sm text-muted-foreground hover:underline">
          go to query →
        </Link>
      </div>

      <ConnectionList />
    </div>
  );
}

function Stat({ title, value, hint }: { title: string; value: string | number; hint: string }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{title}</CardDescription>
        <CardTitle className="text-2xl tabular-nums">{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}
