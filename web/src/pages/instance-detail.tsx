import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Database, ExternalLink, KeyRound, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { useAction } from "@/hooks/use-action";
import { useApp } from "@/app/app-provider";
import { api, type Connection } from "@/lib/api";
import { FAMILY, familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";

// InstanceDetailPage is the cross-family home of one instance: its common
// information and the lifecycle actions. The changes for an instance live on
// the DML/DDL queues, not here.
export default function InstanceDetailPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { isAdmin, reload } = useApp();
  const [instance, setInstance] = useState<Connection>();
  const [newPassword, setNewPassword] = useState("");
  const [error, setError] = useState<string>();

  useEffect(() => {
    let cancelled = false;
    setInstance(undefined);
    setError(undefined);
    api.instances
      .get(id)
      .then((c) => {
        if (!cancelled) setInstance(c);
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

  const rotate = useAction(async () => {
    await api.instances.rotatePassword(id, newPassword);
    setNewPassword("");
    toast.success("password rotated");
  });

  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <Empty
          icon={Database}
          title="Instance not found"
          description={error}
          action={
            <Button size="sm" render={<Link to={PAGE_ROUTES.DASHBOARD} />}>
              Back to Instances
            </Button>
          }
        />
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
          <p className="mt-4 border-t border-border pt-3 text-xs text-muted-foreground">
            Driver, host, database and user are fixed once registered. Only the password can be rotated.
          </p>
          {isAdmin && (
            <div className="mt-3 flex items-end gap-2">
              <div className="grid flex-1 gap-1.5">
                <Label htmlFor="rotate-password">New password</Label>
                <Input
                  id="rotate-password"
                  type="password"
                  value={newPassword}
                  placeholder="rotate the stored password"
                  onChange={(e) => setNewPassword(e.target.value)}
                />
              </div>
              <Button
                size="sm"
                variant="outline"
                onClick={() => void rotate.run()}
                disabled={rotate.loading || newPassword === ""}
              >
                <KeyRound /> Rotate
              </Button>
            </div>
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