import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, ExternalLink, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAction } from "@/hooks/use-action";
import { useApp } from "@/app/app-provider";
import { api, type Change, type Connection } from "@/lib/api";
import { FAMILY, familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

// InstanceDetailPage is the cross-family home of one instance: identity, the
// actions that used to live in the dashboard modal, and its changes grouped by
// kind. The id is stable, so this URL survives renames (which are unsupported).
const STATUS_TONE: Record<string, string> = {
  applied: "text-emerald-500",
  failed: "text-destructive",
  rejected: "text-muted-foreground",
  approved: "text-sky-500",
  validated: "text-sky-500",
  unverified: "text-amber-500",
};

const CHANGE_GROUPS: { kind: string; label: string }[] = [
  { kind: "dml", label: "DML" },
  { kind: "ddl", label: "DDL" },
  { kind: "mongo", label: "Requests" },
];

function changeHref(c: Change) {
  const id = c.id ?? "";
  switch (c.kind) {
    case "dml":
      return PAGE_ROUTES.SQL.DML_DETAIL(id);
    case "ddl":
      return PAGE_ROUTES.SQL.DDL_DETAIL(id);
    default:
      return PAGE_ROUTES.MONGODB.REQUEST_DETAIL(id);
  }
}

export default function InstanceDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { isAdmin, reload } = useApp();
  const [instance, setInstance] = useState<Connection>();
  const [changes, setChanges] = useState<Change[]>([]);
  const [error, setError] = useState<string>();

  useEffect(() => {
    let cancelled = false;
    setInstance(undefined);
    setError(undefined);
    Promise.all([api.instances.get(id), api.instances.changes(id)])
      .then(([c, ch]) => {
        if (cancelled) return;
        setInstance(c);
        setChanges(ch.changes ?? []);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const family = instance?.family as FamilyKey | undefined;
  const name = instance?.name ?? "";

  const test = useAction(async () => {
    if (!family) return;
    await api.family(family).test(name);
    toast.success(`${name}: reachable`);
  });

  const remove = useAction(async () => {
    if (!family) return;
    await api.family(family).remove(name);
    toast.success(`deleted ${name}`);
    await reload(family);
    navigate(PAGE_ROUTES.DASHBOARD);
  });

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <BackLink />
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Instance not found</CardTitle>
            <CardDescription>{error}</CardDescription>
          </CardHeader>
        </Card>
      </div>
    );
  }

  if (!instance || !family) {
    return (
      <div className="flex flex-col gap-4">
        <BackLink />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const meta = familyByKey[family];
  const Icon = meta.icon;
  const databaseLabel =
    family === FAMILY.SQL ? "Database" : family === FAMILY.MONGODB ? "Database (auth)" : "DB index";

  return (
    <div className="flex flex-col gap-4">
      <BackLink />

      <Card>
        <CardHeader className="gap-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="flex items-center gap-3">
              <span className="flex size-9 items-center justify-center rounded-md bg-accent text-muted-foreground">
                <Icon className="size-4" />
              </span>
              <div>
                <CardTitle className="flex items-center gap-2 text-base">
                  {instance.name}
                  <Badge variant="secondary">{instance.driver}</Badge>
                </CardTitle>
                <CardDescription>{meta.label}</CardDescription>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Button size="sm" variant="outline" render={<Link to={meta.page} />}>
                <ExternalLink /> {meta.openLabel}
              </Button>
              <Button size="sm" variant="outline" onClick={() => void test.run()} disabled={test.loading}>
                <Zap /> Test
              </Button>
              {isAdmin && (
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-destructive"
                  onClick={() => void remove.run()}
                  disabled={remove.loading}
                >
                  <Trash2 /> Delete
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-[8rem_1fr] gap-y-2 text-sm sm:grid-cols-[10rem_1fr]">
            <dt className="text-muted-foreground">Driver</dt>
            <dd>
              <Badge variant="secondary">{instance.driver}</Badge>
            </dd>
            <dt className="text-muted-foreground">Endpoint</dt>
            <dd className="font-mono text-xs">
              {instance.host}:{instance.port}
            </dd>
            <dt className="text-muted-foreground">{databaseLabel}</dt>
            <dd>{instance.database}</dd>
            {instance.user && (
              <>
                <dt className="text-muted-foreground">User</dt>
                <dd>{instance.user}</dd>
              </>
            )}
            <dt className="text-muted-foreground">Registered</dt>
            <dd className="text-xs">
              {instance.createdAt ? new Date(instance.createdAt * 1000).toLocaleString() : "—"}
            </dd>
          </dl>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Changes</CardTitle>
          <CardDescription>{changes.length} recorded for this instance</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {changes.length === 0 ? (
            <p className="text-sm text-muted-foreground">no changes recorded yet</p>
          ) : (
            CHANGE_GROUPS.map(({ kind, label }) => {
              const rows = changes.filter((c) => c.kind === kind);
              if (rows.length === 0) return null;
              return (
                <div key={kind} className="flex flex-col gap-2">
                  <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                    {label} · {rows.length}
                  </p>
                  <div className="rounded-md border border-border">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Status</TableHead>
                          <TableHead>Notes</TableHead>
                          <TableHead>Created</TableHead>
                          <TableHead className="w-[80px]" />
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {rows.map((c) => (
                          <TableRow key={c.id}>
                            <TableCell>
                              <span className={cn("text-xs font-medium", STATUS_TONE[c.status ?? ""])}>
                                {c.status}
                              </span>
                            </TableCell>
                            <TableCell className="max-w-[28rem] truncate text-sm">{c.notes}</TableCell>
                            <TableCell className="text-xs text-muted-foreground">
                              {c.createdAt ? new Date(c.createdAt * 1000).toLocaleString() : "—"}
                            </TableCell>
                            <TableCell>
                              <Button size="xs" variant="outline" render={<Link to={changeHref(c)} />}>
                                View
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              );
            })
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function BackLink() {
  return (
    <Button size="sm" variant="ghost" className="w-fit" render={<Link to={PAGE_ROUTES.DASHBOARD} />}>
      <ArrowLeft /> Dashboard
    </Button>
  );
}
