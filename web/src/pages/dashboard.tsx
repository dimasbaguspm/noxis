import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { debounce, title } from "radash";
import { CircleCheck, CircleX, Database, Loader2, Search, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";
import { useConnectionStates } from "@/hooks/use-connection-states";
import { useChanges } from "@/hooks/use-data";
import { api } from "@/lib/api";

export default function DashboardPage() {
  const { connections, reload } = useApp();
  const { changes } = useChanges();
  const { states, connected } = useConnectionStates();

  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);

  const test = useAction(async (name: string) => {
    await api.connections.test(name);
    toast.success(`${name}: reachable`);
  });
  const remove = useAction(async (name: string) => {
    await api.connections.remove(name);
    toast.success(`deleted ${name}`);
    await reload();
  });

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return connections;
    return connections.filter((c) =>
      [c.name, c.driver, c.host, c.database, c.user].some((v) => (v ?? "").toLowerCase().includes(needle)),
    );
  }, [connections, query]);

  const ddl = changes.filter((c) => c.kind === "ddl" && c.status === "applied").length;
  const dml = changes.filter((c) => c.kind === "dml" && c.status === "applied").length;
  const failed = changes.filter((c) => c.status === "failed").length;
  const finished = changes.filter((c) => c.status === "applied" || c.status === "failed").length;
  const errorRate = finished === 0 ? 0 : (failed / finished) * 100;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-lg font-semibold">Dashboard</h2>
        <span className="flex-1" />
        <ConnectionDialog trigger="New connection" />
      </div>

      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Stat title="Instances connected" value={connections.length} hint="registered & verified" />
        <Stat title="DDL executed" value={ddl} hint="schema changes applied" />
        <Stat title="DML executed" value={dml} hint="data changes applied" />
        <Stat
          title="Error rate"
          value={`${errorRate.toFixed(1)}%`}
          hint={`${failed} failed of ${finished} finished`}
          tone={failed > 0 ? "text-destructive" : "text-emerald-500"}
        />
      </div>

      <Card>
        <CardHeader className="gap-3">
          <div>
            <CardTitle className="text-base">Instances</CardTitle>
            <CardDescription>
              {connections.length} registered · reachability streamed live
              {connected ? "" : " (connecting…)"}
            </CardDescription>
          </div>
          <div className="relative max-w-md">
            <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-7"
              placeholder="search instances (name, host, database…)"
              value={filter}
              onChange={(e) => {
                setFilter(e.target.value);
                void setFilterDebounced(e.target.value);
              }}
            />
          </div>
        </CardHeader>
        <CardContent>
          {visible.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {connections.length === 0 ? "no connections yet — add one from the header" : "no match"}
            </p>
          ) : (
            <div className="rounded-md border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{title("name")}</TableHead>
                    <TableHead className="w-[110px]">{title("state")}</TableHead>
                    <TableHead>{title("driver")}</TableHead>
                    <TableHead>{title("endpoint")}</TableHead>
                    <TableHead>{title("database")}</TableHead>
                    <TableHead className="w-[200px]">{title("actions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.map((c) => (
                    <TableRow key={c.name}>
                      <TableCell>
                        <Link to="/query" className="flex items-center gap-2 text-sm hover:underline">
                          <Database className="size-3.5 text-muted-foreground" />
                          {c.name}
                        </Link>
                      </TableCell>
                      <TableCell>
                        <StateCell state={states[c.name ?? ""]} />
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">{c.driver}</Badge>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {c.user}@{c.host}:{c.port}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{c.database}</TableCell>
                      <TableCell className="flex gap-1.5">
                        <Button size="xs" variant="outline" onClick={() => void test.run(c.name ?? "")}>
                          <Zap /> Test
                        </Button>
                        <Button size="xs" variant="ghost" onClick={() => void remove.run(c.name ?? "")}>
                          <Trash2 /> Delete
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
    </div>
  );
}

// StateCell renders the streamed reachability of one row — only the rows the
// dashboard shows are rendered, the stream carries them all.
function StateCell({ state }: { state?: { ok: boolean; latencyMs: number; error?: string } }) {
  if (!state) {
    return (
      <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <Loader2 className="size-3.5 animate-spin" /> checking
      </span>
    );
  }
  if (!state.ok) {
    return (
      <span className="flex items-center gap-1.5 text-xs text-destructive" title={state.error}>
        <CircleX className="size-3.5" /> unreachable
      </span>
    );
  }
  return (
    <span className="flex items-center gap-1.5 text-xs text-emerald-500">
      <CircleCheck className="size-3.5" /> reachable
      <span className="text-muted-foreground">{state.latencyMs} ms</span>
    </span>
  );
}

function Stat({
  title,
  value,
  hint,
  tone,
}: {
  title: string;
  value: string | number;
  hint: string;
  tone?: string;
}) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{title}</CardDescription>
        <CardTitle className={`text-2xl tabular-nums ${tone ?? ""}`}>{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}
