import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { debounce, title } from "radash";
import { Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { useApp } from "@/app/app-provider";
import { type Connection } from "@/lib/api";
import { FAMILIES, familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";

// One row per instance, whichever family it belongs to: the table shows only
// what they have in common, and the detail page carries the rest.
interface Instance extends Connection {
  family: FamilyKey;
}

const ALL_DRIVERS = "all";

export default function DashboardPage() {
  const { registry, isAdmin, drivers } = useApp();

  const instances = useMemo<Instance[]>(
    () => FAMILIES.flatMap((family) => registry[family.key].connections.map((c) => ({ ...c, family: family.key }))),
    [registry],
  );

  const [filter, setFilter] = useState("");
  const [query, setQuery] = useState("");
  const setFilterDebounced = useMemo(() => debounce({ delay: 200 }, (v: string) => setQuery(v)), []);
  const [driver, setDriver] = useState(ALL_DRIVERS);

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return instances.filter((c) => {
      if (driver !== ALL_DRIVERS && c.driver !== driver) return false;
      if (!needle) return true;
      return [c.name, c.driver, c.host, familyByKey[c.family].label].some((v) =>
        (v ?? "").toLowerCase().includes(needle),
      );
    });
  }, [instances, query, driver]);

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="gap-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <CardTitle className="text-base">Instances</CardTitle>
              <CardDescription>{instances.length} registered</CardDescription>
            </div>
            {isAdmin && <ConnectionDialog trigger="New connection" wizard />}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-7"
                placeholder="search by name"
                value={filter}
                onChange={(e) => {
                  setFilter(e.target.value);
                  void setFilterDebounced(e.target.value);
                }}
              />
            </div>
            <Select value={driver} onValueChange={(v) => setDriver(String(v))}>
              <SelectTrigger className="w-[190px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_DRIVERS}>All drivers</SelectItem>
                {drivers.map((d) => (
                  <SelectItem key={d.driver} value={d.driver}>
                    {d.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
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
                    <TableHead>{title("driver")}</TableHead>
                    <TableHead className="w-[110px]">{title("action")}</TableHead>
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
                      <TableCell>
                        <Badge variant="secondary">{c.driver}</Badge>
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
