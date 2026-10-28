import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import {
  Link,
  Navigate,
  Outlet,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
  useOutletContext,
} from "react-router-dom";
import { Database, KeyRound, MoreHorizontal, Trash2, Zap } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CardDescription } from "@/components/ui/card";
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
import type { InstanceTable } from "@/components/instance/tables-tab";
import { useAction } from "@/hooks/use-action";
import { useInstanceQuery, useRotatePasswordMutation } from "@/hooks/use-instances";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { useAuth } from "@/app/auth-provider";
import { api, type Connection } from "@/lib/api";
import { useCrumbLabel } from "@/app/crumb-provider";
import { queryClient, queryKeys } from "@/lib/query-client";
import { FAMILY, familyByKey, type FamilyKey } from "@/lib/families";
import { PAGE_ROUTES } from "@/constants/routes";
import { cn } from "@/lib/utils";

export type InstanceTab = "detail" | "tables" | "access" | "erd" | "documents" | "keys";

export interface InstanceContextValue {
  instance: Connection;
  family: FamilyKey;
  canRead: boolean;
  canAdmin: boolean;
  tables: InstanceTable[];
  schemaPending: boolean;
  schemas: string[];
  selectedSchema: string;
  updateSchema: (schema: string) => void;
}

export function useInstanceContext() {
  return useOutletContext<InstanceContextValue>();
}

const TABS: Record<InstanceTab, string> = {
  detail: "Detail",
  tables: "Tables",
  access: "Access",
  erd: "ERD",
  documents: "Documents",
  keys: "Keys",
};

// InstanceLayout is the cross-family home of one instance: shared header, routed tabs and the lifecycle actions.
export default function InstanceLayout() {
  const { id = "" } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const { isSuper, isInstanceAdmin, canAccess } = useAuth();
  const [rotateOpen, setRotateOpen] = useState(false);
  const detail = useInstanceQuery(id);
  const instance = detail.data;
  // the trail names the instance instead of its id once the detail lands
  useCrumbLabel(PAGE_ROUTES.INSTANCE_DETAIL(id), instance?.name);

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
        comment: table.comment,
        columns: (table.columns ?? []).map((col) => ({
          name: col.name ?? "",
          type: col.type ?? "",
          nullable: col.nullable ?? false,
        })),
        indexes: table.indexes,
        estimatedRows: table.estimatedRows,
        sizeBytes: table.sizeBytes,
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
    toast.success(`Deleted ${name}`);
    void queryClient.invalidateQueries({ queryKey: queryKeys.instances.all });
    navigate(PAGE_ROUTES.INSTANCES);
  });

  const rotate = useRotatePasswordMutation();
  const {
    register,
    handleSubmit,
    reset: resetPassword,
    formState: { errors },
  } = useForm<{ password: string }>({
    defaultValues: { password: "" },
    resolver: zodResolver(z.object({ password: z.string().min(1, "New password is required") })),
  });
  const onRotate = handleSubmit(async ({ password }) => {
    await rotate.mutateAsync({ id, password });
    resetPassword();
    setRotateOpen(false);
    toast.success("Password rotated");
  });

  const schemas = useMemo(() => [...new Set(tables.map((t) => t.schema))].sort(), [tables]);
  const schemaParam = params.get("schema") ?? "";
  const selectedSchema = schemas.includes(schemaParam) ? schemaParam : "";

  function updateSchema(next: string) {
    const query = new URLSearchParams(params);
    if (next) query.set("schema", next);
    else query.delete("schema");
    setParams(query, { replace: true });
  }

  if (detail.error) {
    return (
      <div className="flex flex-col gap-4">
        <Empty
          icon={Database}
          title="Instance not found"
          description={
            detail.error instanceof Error ? detail.error.message : "The instance request failed"
          }
          action={
            <Button size="sm" render={<Link to={PAGE_ROUTES.INSTANCES} />}>
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

  const visibleTabs: { id: InstanceTab; label: string }[] = [
    { id: "detail", label: TABS.detail },
    ...(family === FAMILY.SQL && canRead
      ? [
          { id: "tables" as const, label: TABS.tables },
          { id: "erd" as const, label: TABS.erd },
        ]
      : []),
    ...(family === FAMILY.MONGODB && canRead
      ? [{ id: "documents" as const, label: TABS.documents }]
      : []),
    ...(family === FAMILY.MEMCACHE && canRead ? [{ id: "keys" as const, label: TABS.keys }] : []),
    ...(canAdmin ? [{ id: "access" as const, label: TABS.access }] : []),
  ];
  const base = PAGE_ROUTES.INSTANCE_DETAIL(instance.id ?? id);
  const leaf = location.pathname.slice(base.length).replace(/^\//, "");
  const activeId = (leaf || "detail") as InstanceTab;
  if (!visibleTabs.some((t) => t.id === activeId)) {
    return <Navigate to={base} replace />;
  }

  const meta = familyByKey[family];
  const Icon = meta.icon;
  const context: InstanceContextValue = {
    instance,
    family,
    canRead,
    canAdmin,
    tables,
    schemaPending: schema.isPending,
    schemas,
    selectedSchema,
    updateSchema,
  };

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
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button size="sm" variant="outline" aria-label="Instance actions" />}
            >
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
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={() => void remove.run()}
                    disabled={remove.loading}
                  >
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
            aria-selected={activeId === item.id}
            to={item.id === "detail" ? base : PAGE_ROUTES.INSTANCE_TAB(instance.id ?? id, item.id)}
            className={cn(
              "-mb-px border-b-2 border-transparent px-3 py-2 text-sm text-muted-foreground hover:text-foreground",
              activeId === item.id && "border-foreground text-foreground",
            )}
          >
            {item.label}
          </Link>
        ))}
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
        <Outlet context={context} />
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
              Store a new password for {name}. The old one stops working as soon as the change is
              saved.
            </SheetDescription>
          </SheetHeader>
          <SheetBody>
            <form id="rotate-password" onSubmit={onRotate}>
              <Field>
                <FieldLabel htmlFor="rotate-password-input">New password</FieldLabel>
                <Input
                  id="rotate-password-input"
                  type="password"
                  placeholder="The new stored password"
                  aria-invalid={!!errors.password}
                  {...register("password")}
                />
                <FieldError>
                  {errors.password?.message ??
                    (rotate.isError
                      ? rotate.error instanceof Error
                        ? rotate.error.message
                        : "rotation failed"
                      : undefined)}
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
