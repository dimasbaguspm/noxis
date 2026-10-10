import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { title } from "radash";
import { Boxes, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Pagination } from "@/components/ui/pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { useAuth } from "@/app/auth-provider";
import { useDriversQuery } from "@/hooks/use-config";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import { useDebounce } from "@/hooks/use-debounce";
import { familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";

// One row per instance, whichever family it belongs to: the table shows only
// what they have in common, and the detail page carries the rest.
const ALL_DRIVERS = "all";
const PER_PAGE = 20;
const SKELETON_ROWS = [1, 2, 3, 4, 5, 6];

// DashboardPage filters through the instances API and pages the result, so the
// server stays the source of truth for what the caller may see.
export default function DashboardPage() {
  const { isAdmin } = useAuth();
  const { data: drivers = [] } = useDriversQuery();
  const [filter, setFilter] = useState("");
  const [driver, setDriver] = useState(ALL_DRIVERS);
  const [page, setPage] = useState(1);
  const query = useDebounce(filter);

  // a changed filter starts back at the first page
  useEffect(() => setPage(1), [query, driver]);

  const list = usePagedInstancesQuery({
    name: query.trim() || undefined,
    driver: driver === ALL_DRIVERS ? undefined : driver,
    page,
    perPage: PER_PAGE,
  });
  const instances = (list.data?.instances ?? []).map((c) => ({ ...c, family: c.family as FamilyKey }));
  const total = list.data?.total ?? 0;
  const loading = list.isPending || list.isFetching;

  const filtering = filter.trim() !== "" || driver !== ALL_DRIVERS;

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="gap-3">
          <CardTitle className="text-base">Instances</CardTitle>
          <CardDescription>
            {loading ? "loading…" : filtering ? `${total} shown` : `${total} registered`}
          </CardDescription>
          <div className="flex flex-wrap items-center gap-2">
            <div className="relative min-w-56 flex-1">
              <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-7"
                placeholder="search by name"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
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
      </Card>

      {isAdmin && (
        <div className="flex justify-end">
          <ConnectionDialog trigger="New connection" wizard />
        </div>
      )}

      <Card>
        <CardContent>
          {loading ? (
            <div className="flex flex-col gap-2">
              {SKELETON_ROWS.map((n) => (
                <Skeleton key={n} className="h-11 w-full" />
              ))}
            </div>
          ) : total === 0 ? (
            <Empty
              icon={Boxes}
              title={filtering ? "no match" : "none registered yet"}
              description={filtering ? "try another name or driver" : "register an instance to see it here"}
            />
          ) : (
            <>
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
                    {instances.map((c) => (
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
              {total > PER_PAGE && (
                <Pagination page={page} perPage={PER_PAGE} total={total} onChange={setPage} className="mt-4" />
              )}
            </>
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