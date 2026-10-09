import { useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { debounce, title } from "radash";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { useApp } from "@/app/app-provider";
import { useChanges } from "@/hooks/use-data";
import { type Connection } from "@/lib/api";
import { FAMILIES, familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

// One row per instance, whichever family it belongs to: the table shows only
// what both have in common, and the detail dialog carries the rest.
interface Instance extends Connection {
  family: FamilyKey;
}

export default function DashboardPage() {
  const { registry, isAdmin } = useApp();
  const { changes } = useChanges();

  const instances = useMemo<Instance[]>(
    () => FAMILIES.flatMap((family) => registry[family.key].connections.map((c) => ({ ...c, family: family.key }))),
    [registry],
  );

  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);

  const ddl = changes.filter((c) => c.kind === "ddl" && c.status === "applied").length;
  const dml = changes.filter((c) => c.kind === "dml" && c.status === "applied").length;
  const failed = changes.filter((c) => c.status === "failed").length;
  const finished = changes.filter((c) => c.status === "applied" || c.status === "failed").length;
  const errorRate = finished === 0 ? 0 : (failed / finished) * 100;

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return instances;
    return instances.filter((c) =>
      [c.name, c.driver, c.host, c.database, c.user, familyByKey[c.family].label].some((v) =>
        (v ?? "").toLowerCase().includes(needle),
      ),
    );
  }, [instances, query]);

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <TotalTile
          all={instances.length}
          sql={registry.sql.connections.length}
          memcache={registry.memcache.connections.length}
          mongo={registry.mongodb.connections.length}
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
              <CardDescription>{instances.length} registered</CardDescription>
            </div>
            {isAdmin && <ConnectionDialog trigger="New connection" wizard />}
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
                    <TableHead>{title("endpoint")}</TableHead>
                    <TableHead className="w-[110px]">{title("detail")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {visible.map((c) => (
                    <TableRow key={`${c.family}:${c.name}`}>
                      <TableCell>
                        <span className="flex items-center gap-2 text-sm">
                          <FamilyIcon family={c.family} className="size-3.5 text-muted-foreground" />
                          {c.name}
                        </span>
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {c.user ? `${c.user}@` : ""}
                        {c.host}:{c.port}
                      </TableCell>
                      <TableCell>
                        {c.id ? (
                          <Button size="xs" variant="outline" render={<Link to={PAGE_ROUTES.INSTANCE_DETAIL(c.id)} />}>
                            Detail
                          </Button>
                        ) : (
                          <span className="text-xs text-muted-foreground">—</span>
                        )}
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

// FamilyIcon renders a family's glyph without naming a component per family.
function FamilyIcon({ family, className }: { family: FamilyKey; className?: string }) {
  const Icon = familyByKey[family].icon;
  return <Icon className={className} />;
}

// TotalTile is one card holding every count: swipe (or click) between All,
// SQL and Memory Cache. The slides are a scroll-snapped track, so a trackpad
// swipe works and the tabs stay in sync.
function TotalTile({
  all,
  sql,
  memcache,
  mongo,
}: {
  all: number;
  sql: number;
  memcache: number;
  mongo: number;
}) {
  const slides = [
    { label: "All", value: all },
    { label: "SQL", value: sql },
    { label: "Memory Cache", value: memcache },
    { label: "MongoDB", value: mongo },
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
