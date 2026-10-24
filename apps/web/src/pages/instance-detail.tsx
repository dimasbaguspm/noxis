import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { Database, ExternalLink, KeyRound, MoreHorizontal, Play, ShieldCheck, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty } from "@/components/ui/empty";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { RequestAccessDialog } from "@/components/access/request-access-dialog";
import { InstancePermissions } from "@/components/access/instance-permissions";
import { CacheBrowser } from "@/components/instance/cache-browser";
import { InstanceErd } from "@/components/instance/erd-tab";
import { MongoExplorer } from "@/components/instance/mongo-explorer";
import { TablesTab, type InstanceTable } from "@/components/instance/tables-tab";
import { useAction } from "@/hooks/use-action";
import { useConnections } from "@/hooks/use-connections";
import { useInstanceQuery, useRotatePasswordMutation } from "@/hooks/use-instances";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { useAuth } from "@/app/auth-provider";
import { api } from "@/lib/api";
import { queryClient } from "@/lib/query-client";
import { FAMILY, familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

export type InstanceTab = "detail" | "tables" | "access" | "erd" | "documents" | "keys";

const TABS = {
  detail: "Detail",
  tables: "Tables",
  access: "Access",
  erd: "ERD",
  documents: "Documents",
  keys: "Keys",
} as const;

// InstanceDetailPage is the cross-family home of one instance: shared information, family tabs and the lifecycle actions.
export default function InstanceDetailPage({ tab = "detail" }: { tab?: InstanceTab }) {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const { isSuper, isInstanceAdmin, canAccess } = useAuth();
  const [rotateOpen, setRotateOpen] = useState(false);
  const detail = useInstanceQuery(id);
  const instance = detail.data;
  const error = detail.error instanceof Error ? detail.error.message : undefined;

  const family = instance?.family as FamilyKey | undefined;
  const name = instance?.name ?? "";
  const canRead = canAccess(name, "read");
  const canAdmin = isSuper || isInstanceAdmin(name);
  const schema = useConnectionSchemaQuery(family === FAMILY.SQL && canRead ? name : undefined);

  const tables = useMemo<InstanceTable[]>(
    () =>
      (schema.data ?? []).map((table) => ({
        id: table.id,
        schema: table.schema ?? "",
        name: table.name ?? "",
        columns: (table.columns ?? []).map((col) => ({
          name: col.name ?? "",
          type: col.type ?? "",
          nullable: col.nullable ?? false,
        })),
        indexes: table.indexes,
        estimatedRows: table.estimatedRows,
      })),
    [schema.data],
  );

  const test = useAction(async () => {
    if (!family) return;
    await api.family(family).test(name);
    toast.success(`${name}: reachable`);
  });

  const remove = useAction(async () => {
    if (!family) return;
    await api.family(family).remove(name);
    toast.success(`deleted ${name}`);
    void queryClient.invalidateQueries({ queryKey: ["instances"] });
    navigate(PAGE_ROUTES.DASHBOARD);
  });

  const rotate = useRotatePasswordMutation();
  const {
    register,
    handleSubmit,
    reset: resetPassword,
    formState: { errors },
  } = useForm<{ password: string }>({
    defaultValues: { password: "" },
    resolver: zodResolver(
      z.object({ password: z.string().min(1, "new password is required") }),
    ),
  });
  const onRotate = handleSubmit(async ({ password }) => {
    await rotate.mutateAsync({ id, password });
    resetPassword();
    setRotateOpen(false);
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
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  const schemas = [...new Set(tables.map((t) => t.schema))].sort();
  const schemaParam = params.get("schema") ?? "";
  const selectedSchema = schemas.includes(schemaParam) ? schemaParam : "";
  const visibleTabs = [
    { id: "detail", label: TABS.detail },
    ...(family === FAMILY.SQL && canRead
      ? [
          { id: "tables", label: TABS.tables },
          { id: "erd", label: TABS.erd },
        ]
      : []),
    ...(family === FAMILY.MONGODB && canRead ? [{ id: "documents", label: TABS.documents }] : []),
    ...(family === FAMILY.MEMCACHE && canRead ? [{ id: "keys", label: TABS.keys }] : []),
    ...(canAdmin ? [{ id: "access", label: TABS.access }] : []),
  ];
  const active = visibleTabs.find((t) => t.id === tab) ?? visibleTabs[0];

  function updateSchema(next: string) {
    const query = new URLSearchParams(params);
    if (next) query.set("schema", next);
    else query.delete("schema");
    setParams(query, { replace: true });
  }

  const meta = familyByKey[family];
  const Icon = meta.icon;
  const fileDriver = instance.driver === "sqlite";
  const databaseLabel = fileDriver
    ? "File path"
    : family === FAMILY.SQL
      ? "Database"
      : family === FAMILY.MONGODB
        ? "Database (auth)"
        : "DB index";

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-accent text-muted-foreground">
            <Icon className="size-4" />
          </span>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="truncate text-base font-medium">{instance.name}</h1>
              <Badge variant="secondary">{instance.driver}</Badge>
            </div>
            <CardDescription>{meta.label}</CardDescription>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {!isSuper && <RequestAccessDialog connection={instance.name} />}
          {canRead && <OpenFamily family={family} name={instance.name} />}
          {canAdmin && (
            <Button
              size="sm"
              variant="outline"
              render={<Link to={PAGE_ROUTES.INSTANCE_DASHBOARD(instance.id ?? "")} />}
            >
              <ShieldCheck /> Dashboard
            </Button>
          )}
          <DropdownMenu>
            <DropdownMenuTrigger render={<Button size="sm" variant="outline" aria-label="Instance actions" />}>
              <MoreHorizontal />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => void test.run()} disabled={test.loading}>
                <Zap /> Test
              </DropdownMenuItem>
              {isSuper && (
                <DropdownMenuItem onClick={() => setRotateOpen(true)}>
                  <KeyRound /> Rotate password
                </DropdownMenuItem>
              )}
              {isSuper && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem variant="destructive" onClick={() => void remove.run()} disabled={remove.loading}>
                    <Trash2 /> Delete
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div role="tablist" className="flex flex-wrap gap-1 border-b border-border">
        {visibleTabs.map((item) => (
          <Link
            key={item.id}
            role="tab"
            aria-selected={active.id === item.id}
            to={PAGE_ROUTES.INSTANCE_TAB(instance.id ?? id, item.id)}
            className={cn(
              "-mb-px border-b-2 border-transparent px-3 py-2 text-sm text-muted-foreground hover:text-foreground",
              active.id === item.id && "border-foreground text-foreground",
            )}
          >
            {item.label}
          </Link>
        ))}
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
      {active.id === "detail" && (
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader className="gap-1">
              <CardTitle className="text-base">Details</CardTitle>
              <CardDescription>Driver, endpoint and registration metadata.</CardDescription>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-[8rem_1fr] gap-y-2 text-sm sm:grid-cols-[10rem_1fr]">
                <dt className="text-muted-foreground">Driver</dt>
                <dd>
                  <Badge variant="secondary">{instance.driver}</Badge>
                </dd>
                {!fileDriver && (
                  <>
                    <dt className="text-muted-foreground">Endpoint</dt>
                    <dd className="font-mono text-xs">
                      {instance.host}:{instance.port}
                    </dd>
                  </>
                )}
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
                  {instance.createdBy ? ` · by ${instance.createdBy}` : ""}
                </dd>
              </dl>
              <P className="mt-4 border-t border-border pt-3">
                Driver, host, database and user are fixed once registered. Only the password can be rotated.
              </P>
            </CardContent>
          </Card>

          {family === FAMILY.SQL && canRead && (
            <Card>
              <CardHeader className="gap-1">
                <CardTitle className="text-base">Schemas</CardTitle>
                <CardDescription>Available schemas and how many tables each holds.</CardDescription>
              </CardHeader>
              <CardContent>
                {schema.isPending ? (
                  <Skeleton className="h-20 w-full" />
                ) : schemas.length === 0 ? (
                  <p className="text-xs text-muted-foreground">No tables in this instance.</p>
                ) : (
                  <div className="flex gap-2 overflow-x-auto pb-1">
                    {schemas.map((s) => {
                      const count = tables.filter((t) => t.schema === s).length;
                      return (
                        <Link
                          key={s}
                          to={`${PAGE_ROUTES.INSTANCE_TAB(instance.id ?? id, "tables")}?schema=${encodeURIComponent(s)}`}
                          className="min-w-32 shrink-0 rounded-lg border p-3 text-left hover:bg-accent/50"
                        >
                          <div className="truncate font-mono text-xs">{s}</div>
                          <div className="mt-1 text-lg font-medium tabular-nums">{count}</div>
                          <div className="text-[10px] text-muted-foreground">table(s)</div>
                        </Link>
                      );
                    })}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {active.id === "tables" && (
        <TablesTab
          tables={tables}
          loading={schema.isPending}
          schema={selectedSchema}
          onSchema={updateSchema}
        />
      )}

      {active.id === "erd" && <InstanceErd connection={instance.name} />}
      {active.id === "documents" && <MongoExplorer connection={instance.name} />}
      {active.id === "keys" && <CacheBrowser connection={instance.name} />}
      {active.id === "access" && <InstancePermissions instance={instance} readOnly />}
      </div>

      <Sheet
        open={rotateOpen}
        onOpenChange={(open) => {
          setRotateOpen(open);
          if (!open) resetPassword();
        }}
      >
        <SheetContent className="sm:max-w-sm">
          <SheetHeader>
            <SheetTitle>Rotate the password</SheetTitle>
            <SheetDescription>
              Store a new password for {name}. The old one stops working as soon as the change is saved.
            </SheetDescription>
          </SheetHeader>
          <SheetBody>
            <form id="rotate-password" onSubmit={onRotate}>
              <Field>
                <FieldLabel htmlFor="rotate-password-input">New password</FieldLabel>
                <Input
                  id="rotate-password-input"
                  type="password"
                  placeholder="the new stored password"
                  aria-invalid={!!errors.password}
                  {...register("password")}
                />
                <FieldError>
                  {errors.password?.message ??
                    (rotate.isError ? (rotate.error instanceof Error ? rotate.error.message : "rotation failed") : undefined)}
                </FieldError>
              </Field>
            </form>
          </SheetBody>
          <SheetFooter>
            <Button size="sm" type="submit" form="rotate-password" disabled={rotate.isPending}>
              <KeyRound /> Rotate
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

// OpenFamily is the family-specific primary action: the query page takes a connection seed, the other pages reuse the active instance.
function OpenFamily({ family, name }: { family: FamilyKey; name: string }) {
  const { setActive } = useConnections(family);
  const meta = familyByKey[family];
  if (family === FAMILY.SQL) {
    return (
      <Button size="sm" render={<Link to={`${PAGE_ROUTES.SQL.QUERY}?connection=${encodeURIComponent(name)}`} />}>
        <Play /> Query
      </Button>
    );
  }
  return (
    <Button size="sm" render={<Link to={meta.page} onClick={() => setActive(name)} />}>
      <ExternalLink /> {family === FAMILY.MONGODB ? "Explore" : "Browse keys"}
    </Button>
  );
}
