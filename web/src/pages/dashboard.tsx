import { useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { debounce, title } from "radash";
import { Boxes, CircleCheck, CircleX, Database, Loader2, Search, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { useApp } from "@/app/app-provider";
import { useAction } from "@/hooks/use-action";
import { useConnectionStates, type ConnectionState } from "@/hooks/use-connection-states";
import { useChanges } from "@/hooks/use-data";
import { api, type Connection } from "@/lib/api";
import { cn } from "@/lib/utils";

type Family = "sql" | "memcache";

// One row per instance, whichever family it belongs to: the table shows only
// what both have in common, and the detail dialog carries the rest.
interface Instance extends Connection {
  family: Family;
}

const FAMILY_LABEL: Record<Family, string> = { sql: "SQL", memcache: "Memory Cache" };

export default function DashboardPage() {
  const { connections, memcacheConnections, reload, reloadMemcache } = useApp();
  const { changes } = useChanges();
  const { states, connected } = useConnectionStates();

  const instances = useMemo<Instance[]>(
    () => [
      ...connections.map((c) => ({ ...c, family: "sql" as const })),
      ...memcacheConnections.map((c) => ({ ...c, family: "memcache" as const })),
    ],
    [connections, memcacheConnections],
  );

  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);
  const [detail, setDetail] = useState<Instance>();

  const ddl = changes.filter((c) => c.kind === "ddl" && c.status === "applied").length;
  const dml = changes.filter((c) => c.kind === "dml" && c.status === "applied").length;
  const failed = changes.filter((c) => c.status === "failed").length;
  const finished = changes.filter((c) => c.status === "applied" || c.status === "failed").length;
  const errorRate = finished === 0 ? 0 : (failed / finished) * 100;

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return instances;
    return instances.filter((c) =>
      [c.name, c.driver, c.host, c.database, c.user, FAMILY_LABEL[c.family]].some((v) =>
        (v ?? "").toLowerCase().includes(needle),
      ),
    );
  }, [instances, query]);

  const test = useAction(async (c: Instance) => {
    const client = c.family === "memcache" ? api.memcache : api.connections;
    await client.test(c.name ?? "");
    toast.success(`${c.name}: reachable`);
  });

  const remove = useAction(async (c: Instance) => {
    const client = c.family === "memcache" ? api.memcache : api.connections;
    await client.remove(c.name ?? "");
    toast.success(`deleted ${c.name}`);
    await (c.family === "memcache" ? reloadMemcache() : reload());
    setDetail(undefined);
  });

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <TotalTile
          all={instances.length}
          sql={connections.length}
          memcache={memcacheConnections.length}
        />
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
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle className="text-base">Instances</CardTitle>
              <CardDescription>
                {instances.length} registered · reachability streamed live
                {connected ? "" : " (connecting…)"}
              </CardDescription>
            </div>
            <ConnectionDialog trigger="New connection" wizard />
          </div>
          <div className="relative max-w-md">
            <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-7"
              placeholder="search instances (name, host, driver…)"
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
              {instances.length === 0 ? "none registered yet" : "no match"}
            </p>
          ) : (
            <div className="rounded-md border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{title("name")}</TableHead>
                    <TableHead className="w-[110px]">{title("state")}</TableHead>
                    <TableHead>{title("endpoint")}</TableHead>
                    <TableHead className="w-[110px]">{title("detail")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.map((c) => (
                    <TableRow key={`${c.family}:${c.name}`}>
                      <TableCell>
                        <span className="flex items-center gap-2 text-sm">
                          {c.family === "sql" ? (
                            <Database className="size-3.5 text-muted-foreground" />
                          ) : (
                            <Boxes className="size-3.5 text-muted-foreground" />
                          )}
                          {c.name}
                        </span>
                      </TableCell>
                      <TableCell>
                        <StateCell state={states[c.name ?? ""]} />
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {c.user ? `${c.user}@` : ""}
                        {c.host}:{c.port}
                      </TableCell>
                      <TableCell>
                        <Button size="xs" variant="outline" onClick={() => setDetail(c)}>
                          Detail
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

      <Dialog open={detail !== undefined} onOpenChange={(open) => !open && setDetail(undefined)}>
        <DialogContent className="sm:max-w-md">
          {detail && (
            <>
              <DialogHeader>
                <DialogTitle className="flex items-center gap-2">
                  {detail.family === "sql" ? (
                    <Database className="size-4" />
                  ) : (
                    <Boxes className="size-4" />
                  )}
                  {detail.name}
                </DialogTitle>
                <DialogDescription>{FAMILY_LABEL[detail.family]}</DialogDescription>
              </DialogHeader>

              <dl className="grid grid-cols-[8rem_1fr] gap-y-2 text-sm">
                <dt className="text-muted-foreground">Driver</dt>
                <dd>
                  <Badge variant="secondary">{detail.driver}</Badge>
                </dd>
                <dt className="text-muted-foreground">Endpoint</dt>
                <dd className="font-mono text-xs">
                  {detail.host}:{detail.port}
                </dd>
                <dt className="text-muted-foreground">{detail.family === "sql" ? "Database" : "DB index"}</dt>
                <dd>{detail.database}</dd>
                {detail.user && (
                  <>
                    <dt className="text-muted-foreground">User</dt>
                    <dd>{detail.user}</dd>
                  </>
                )}
                <dt className="text-muted-foreground">State</dt>
                <dd>
                  <StateCell state={states[detail.name ?? ""]} />
                </dd>
                <dt className="text-muted-foreground">Registered</dt>
                <dd className="text-xs">
                  {detail.createdAt ? new Date(detail.createdAt * 1000).toLocaleString() : "—"}
                </dd>
              </dl>

              <DialogFooter className="items-center gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  render={
                    <Link to={detail.family === "sql" ? "/query" : "/memcache"} />
                  }
                >
                  {detail.family === "sql" ? "Open in Query" : "Browse keys"}
                </Button>
                <Button size="sm" variant="outline" onClick={() => void test.run(detail)}>
                  <Zap /> Test
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-destructive"
                  onClick={() => void remove.run(detail)}
                >
                  <Trash2 /> Delete
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// TotalTile is one card holding every count: swipe (or click) between All,
// SQL and Memory Cache. The slides are a scroll-snapped track, so a trackpad
// swipe works and the tabs stay in sync.
function TotalTile({ all, sql, memcache }: { all: number; sql: number; memcache: number }) {
  const slides = [
    { label: "All", value: all },
    { label: "SQL", value: sql },
    { label: "Memory Cache", value: memcache },
  ];
  const [active, setActive] = useState(0);
  const track = useRef<HTMLDivElement>(null);

  function go(index: number) {
    setActive(index);
    track.current?.scrollTo({ left: index * (track.current.clientWidth || 0), behavior: "smooth" });
  }

  return (
    <Card>
      <CardHeader className="gap-1.5">
        <div className="flex items-center justify-between gap-2">
          <CardDescription>Instances</CardDescription>
          <div className="flex gap-0.5">
            {slides.map((slide, i) => (
              <button
                key={slide.label}
                onClick={() => go(i)}
                className={cn(
                  "rounded px-1.5 py-0.5 text-[10px]",
                  i === active ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:bg-accent/50",
                )}
              >
                {slide.label}
              </button>
            ))}
          </div>
        </div>
        <div
          ref={track}
          className="flex snap-x snap-mandatory overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
          onScroll={(e) => {
            const el = e.currentTarget;
            if (el.clientWidth > 0) setActive(Math.round(el.scrollLeft / el.clientWidth));
          }}
        >
          {slides.map((slide) => (
            <div key={slide.label} className="w-full shrink-0 snap-center">
              <CardTitle className="text-2xl tabular-nums">{slide.value}</CardTitle>
            </div>
          ))}
        </div>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground">
          {slides[active]?.label} · {all} total registered
        </p>
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
