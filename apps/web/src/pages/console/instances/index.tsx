import { Link } from "react-router-dom";
import { title } from "radash";
import { Boxes, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Pagination } from "@/components/ui/pagination";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ConnectionDialog } from "@/components/connection/connection-dialog";
import { Workspace } from "@/components/layout/workspace";
import { useAuth } from "@/app/auth-provider";
import { useDriversQuery } from "@/hooks/use-config";
import { ALL_DRIVERS, useInstancesUrlParams, usePagedInstancesQuery } from "@/hooks/use-instances";
import { useDebounce } from "@/hooks/use-debounce";
import { capabilitiesOf, capabilityIcon } from "@/lib/console-nav";
import type { DriverInfo } from "@/lib/api";
import { PAGE_ROUTES } from "@/constants/routes";

// One row per instance, whichever driver serves it: the table shows only what they have in common, and the detail page carries the rest.
const PER_PAGE = 20;
const SKELETON_ROWS = [1, 2, 3, 4, 5, 6];

// DashboardPage filters through the instances API and pages the result, so the server stays the source of truth for what the caller may see.
export default function DashboardPage() {
  const { isSuper } = useAuth();
  const { data: drivers = [] } = useDriversQuery();
  const { q: filter, driver, page, setQ: setFilter, setDriver, setPage } = useInstancesUrlParams();
  const query = useDebounce(filter);

  const list = usePagedInstancesQuery({
    name: query.trim() || undefined,
    driver: driver === ALL_DRIVERS ? undefined : driver,
    page,
    perPage: PER_PAGE,
  });
  const instances = list.data?.instances ?? [];
  const total = list.data?.total ?? 0;
  const loading = list.isPending || list.isFetching;

  const filtering = filter.trim() !== "" || driver !== ALL_DRIVERS;

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="instance-search">Search by name</Label>
              <InputGroup>
                <InputGroupAddon>
                  <Search />
                </InputGroupAddon>
                <InputGroupInput
                  id="instance-search"
                  placeholder="Type to filter…"
                  value={filter}
                  onChange={(e) => setFilter(e.target.value)}
                />
              </InputGroup>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Driver</Label>
              <Select value={driver} onValueChange={(v) => setDriver(String(v))}>
                <SelectTrigger className="w-full">
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
            {isSuper && <ConnectionDialog trigger="New connection" wizard />}
          </div>
        }
        right={
          <div className="min-h-0 flex-1 overflow-auto">
            {loading ? (
              <div className="flex flex-col gap-2">
                {SKELETON_ROWS.map((n) => (
                  <Skeleton key={n} className="h-11 w-full" />
                ))}
              </div>
            ) : total === 0 ? (
              <Empty
                icon={Boxes}
                title={filtering ? "No match" : "None registered yet"}
                description={
                  filtering ? "Try another name or driver" : "Register an instance to see it here"
                }
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
                        <TableRow key={`${c.driver}:${c.name}`}>
                          <TableCell>
                            <span className="flex items-center gap-2 text-sm">
                              <DriverIcon
                                driver={c.driver}
                                catalog={drivers}
                                className="size-3.5 text-muted-foreground"
                              />
                              {c.name}
                            </span>
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary">{c.driver}</Badge>
                          </TableCell>
                          <TableCell>
                            {c.id ? (
                              <Button
                                size="xs"
                                variant="outline"
                                render={<Link to={PAGE_ROUTES.INSTANCE_DETAIL(c.id)} />}
                              >
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
              </>
            )}
          </div>
        }
        footer={
          total > 0 ? (
            <>
              <span>
                {total} instance{total === 1 ? "" : "s"}
              </span>
              {total > PER_PAGE && (
                <Pagination page={page} perPage={PER_PAGE} total={total} onChange={setPage} />
              )}
            </>
          ) : undefined
        }
      />
    </div>
  );
}

// DriverIcon renders the glyph of the capability a driver declares, without naming a component per driver; a driver the
// catalog gives no capability falls back to the generic box.
function DriverIcon({
  driver,
  catalog,
  className,
}: {
  driver?: string;
  catalog: DriverInfo[];
  className?: string;
}) {
  const Icon = capabilityIcon(capabilitiesOf(catalog, driver));
  return <Icon className={className} />;
}
