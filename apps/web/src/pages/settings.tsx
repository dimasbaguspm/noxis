import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, ShieldCheck, X } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuth } from "@/app/auth-provider";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import { api, type Connection } from "@/lib/api";
import { familyByKey, type FamilyKey } from "@/lib/families";

const LEVELS = [
  { value: "read", label: "Read" },
  { value: "write", label: "Editor" },
  { value: "admin", label: "Admin" },
] as const;

const SKELETON_ROWS = [1, 2, 3];

// SettingsPage is the RBAC console: per-instance grants binding OIDC groups
// to read/editor/admin. Only global admins reach it (the server enforces the
// same on every grants call); the configured admin group is implicit super
// admin on every instance.
export default function SettingsPage() {
  const { isAdmin } = useAuth();
  const { data, isPending } = usePagedInstancesQuery();

  if (!isAdmin) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Admins only"
          description="Managing access is reserved for global administrators."
        />
      </div>
    );
  }

  const instances = data?.instances ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-base font-medium">Access</h1>
        <p className="text-xs text-muted-foreground">
          Bind OIDC groups to a level per instance. The group in NOXIS_OIDC_ADMIN_GROUP is super
          admin everywhere and needs no grant.
        </p>
      </div>

      {isPending ? (
        <div className="flex flex-col gap-2">
          {SKELETON_ROWS.map((n) => (
            <Skeleton key={n} className="h-24 w-full" />
          ))}
        </div>
      ) : instances.length === 0 ? (
        <Empty title="No instances" description="Register an instance before granting access." />
      ) : (
        instances.map((instance) => <InstanceGrants key={instance.id} instance={instance} />)
      )}
    </div>
  );
}

function InstanceGrants({ instance }: { instance: Connection }) {
  const family = instance.family as FamilyKey | undefined;
  const name = instance.name;
  const queryClient = useQueryClient();
  const { identity } = useAuth();
  const [group, setGroup] = useState("");
  const [level, setLevel] = useState<string>("read");

  const grants = useQuery({
    queryKey: ["grants", family, name],
    queryFn: () => api.grants.list(family as FamilyKey, name),
    enabled: !!family && !!name,
  });
  const known = useQuery({ queryKey: ["config", "groups"], queryFn: () => api.config.groups() });

  const suggestions = useMemo(() => {
    const all = new Set<string>([...(identity?.groups ?? []), ...(known.data?.groups ?? [])]);
    return [...all].sort();
  }, [identity, known.data]);

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: ["grants", family, name] });
    void queryClient.invalidateQueries({ queryKey: ["me"] });
  }

  const add = useMutation({
    mutationFn: () => api.grants.set(family as FamilyKey, name, group.trim(), level),
    onSuccess: () => {
      setGroup("");
      invalidate();
      toast.success(`${group.trim()} can ${level} ${name}`);
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });
  const remove = useMutation({
    mutationFn: (grantGroup: string) => api.grants.remove(family as FamilyKey, name, grantGroup),
    onSuccess: invalidate,
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });

  const meta = family ? familyByKey[family] : undefined;
  const Icon = meta?.icon;

  return (
    <Card>
      <CardHeader className="gap-1">
        <CardTitle className="flex items-center gap-2 text-base">
          {Icon && <Icon className="size-4 text-muted-foreground" />}
          {name}
        </CardTitle>
        <CardDescription>
          {meta?.label ?? family} · {instance.driver}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          {(grants.data?.grants ?? []).length === 0 ? (
            <span className="text-sm text-muted-foreground">no grants yet</span>
          ) : (
            (grants.data?.grants ?? []).map((grant) => (
              <Badge key={grant.group} variant="secondary" className="gap-1.5 pr-1">
                {grant.group}
                <span className="text-muted-foreground">
                  · {LEVELS.find((l) => l.value === grant.level)?.label ?? grant.level}
                </span>
                <button
                  type="button"
                  aria-label={`remove ${grant.group}`}
                  className="rounded-sm p-0.5 hover:bg-background/60"
                  onClick={() => remove.mutate(grant.group)}
                >
                  <X className="size-3" />
                </button>
              </Badge>
            ))
          )}
        </div>

        <form
          className="flex flex-wrap items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (group.trim()) add.mutate();
          }}
        >
          <Input
            className="w-56"
            placeholder="oidc group name"
            list="known-groups"
            value={group}
            onChange={(e) => setGroup(e.target.value)}
          />
          <datalist id="known-groups">
            {suggestions.map((g) => (
              <option key={g} value={g} />
            ))}
          </datalist>
          <Select value={level} onValueChange={(v) => setLevel(String(v))}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {LEVELS.map((l) => (
                <SelectItem key={l.value} value={l.value}>
                  {l.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button type="submit" size="sm" disabled={!group.trim() || add.isPending}>
            <Plus /> Grant
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}