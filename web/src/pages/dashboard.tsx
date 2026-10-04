import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { debounce, title } from "radash";
import { Boxes, CircleCheck, CircleX, Database, KeyRound, Loader2, Search, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";
import { useConnectionStates, type ConnectionState } from "@/hooks/use-connection-states";
import { useChanges } from "@/hooks/use-data";
import { api, type Connection } from "@/lib/api";

// The dashboard keeps the engine families apart: SQL instances and Redis
// (or Valkey) servers are different things, with their own tables and their
// own "new connection" entry point.
export default function DashboardPage() {
  const { connections, redisConnections, reload, reloadRedis } = useApp();
  const { changes } = useChanges();
  const { states, connected } = useConnectionStates();

  const ddl = changes.filter((c) => c.kind === "ddl" && c.status === "applied").length;
  const dml = changes.filter((c) => c.kind === "dml" && c.status === "applied").length;
  const failed = changes.filter((c) => c.status === "failed").length;
  const finished = changes.filter((c) => c.status === "applied" || c.status === "failed").length;
  const errorRate = finished === 0 ? 0 : (failed / finished) * 100;

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
        <Stat title="SQL instances" value={connections.length} hint="registered and verified" />
        <Stat title="Redis / Valkey" value={redisConnections.length} hint="key-value servers" />
        <Stat title="DDL executed" value={ddl} hint="schema changes applied" />
        <Stat title="DML executed" value={dml} hint="data changes applied" />
        <Stat
          title="Error rate"
          value={`${errorRate.toFixed(1)}%`}
          hint={`${failed} failed of ${finished} finished`}
          tone={failed > 0 ? "text-destructive" : "text-emerald-500"}
        />
      </div>

      <InstanceCard
        title="SQL instances"
        description="Postgres, MySQL and MariaDB — the pages under SQL read and change these"
        searchPlaceholder="search instances (name, host, database…)"
        icon={<Database className="size-3.5 text-muted-foreground" />}
        connections={connections}
        states={states}
        streamConnected={connected}
        newConnection={<ConnectionDialog trigger="New connection" />}
        extraColumn="database"
        onTest={api.connections.test}
        onDelete={api.connections.remove}
        onSaved={reload}
        open={{ to: "/query", label: "Open in Query" }}
      />

      <InstanceCard
        title="Redis / Valkey"
        description="Key-value servers: browse keys, inspect values, watch INFO metrics"
        searchPlaceholder="search servers (name, host, db index…)"
        icon={<Boxes className="size-3.5 text-muted-foreground" />}
        connections={redisConnections}
        states={states}
        streamConnected={connected}
        newConnection={<ConnectionDialog family="redis" trigger="New connection" />}
        extraColumn="db index"
        onTest={api.redis.test}
        onDelete={api.redis.remove}
        onSaved={reloadRedis}
        open={{ to: "/redis", label: "Browse keys" }}
      />
    </div>
  );
}

// InstanceCard is one family's table: its own search, its own actions, and the
// reachability the shared stream reports for every connection.
function InstanceCard({
  title: heading,
  description,
  searchPlaceholder,
  icon,
  connections,
  states,
  streamConnected,
  newConnection,
  extraColumn,
  onTest,
  onDelete,
  onSaved,
  open,
}: {
  title: string;
  description: string;
  searchPlaceholder: string;
  icon: React.ReactNode;
  connections: Connection[];
  states: Record<string, ConnectionState>;
  streamConnected: boolean;
  newConnection: React.ReactNode;
  extraColumn: string;
  onTest: (name: string) => Promise<unknown>;
  onDelete: (name: string) => Promise<unknown>;
  onSaved: () => Promise<void>;
  open: { to: string; label: string };
}) {
  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);

  const test = useAction(async (name: string) => {
    await onTest(name);
    toast.success(`${name}: reachable`);
  });
  const remove = useAction(async (name: string) => {
    await onDelete(name);
    toast.success(`deleted ${name}`);
    await onSaved();
  });

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return connections;
    return connections.filter((c) =>
      [c.name, c.driver, c.host, c.database, c.user].some((v) => (v ?? "").toLowerCase().includes(needle)),
    );
  }, [connections, query]);

  return (
    <Card>
      <CardHeader className="gap-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">{heading}</CardTitle>
            <CardDescription>
              {connections.length} registered · reachability streamed live
              {streamConnected ? "" : " (connecting…)"}
            </CardDescription>
          </div>
          {newConnection}
        </div>
        <p className="text-xs text-muted-foreground">{description}</p>
        <div className="relative max-w-md">
          <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-7"
            placeholder={searchPlaceholder}
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
            {connections.length === 0 ? "none registered yet" : "no match"}
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
                  <TableHead>{title(extraColumn)}</TableHead>
                  <TableHead className="w-[230px]">{title("actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visible.map((c) => (
                  <TableRow key={c.name}>
                    <TableCell>
                      <Link to={open.to} className="flex items-center gap-2 text-sm hover:underline" title={open.label}>
                        {icon}
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
                      {c.user ? `${c.user}@` : ""}
                      {c.host}:{c.port}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{c.database}</TableCell>
                    <TableCell className="flex gap-1.5">
                      <Button size="xs" variant="outline" render={<Link to={open.to} />}>
                        <KeyRound /> {open.label}
                      </Button>
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
  );
}

// StateCell renders the streamed reachability of one row — only the rows the
// dashboard shows are rendered, the stream carries them all.
function StateCell({ state }: { state?: ConnectionState }) {
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
