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

const MEMBER_TYPES = [
  { value: "group", label: "Group" },
  { value: "user", label: "User" },
] as const;

const SKELETON_ROWS = [1, 2, 3];

// SettingsPage is the RBAC console: per-instance permissions binding a user (by
// subject) or a group to read/editor/admin. Only global admins reach it (the
// server enforces the same on every permissions call); a global admin is
// implicit super admin on every instance.
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
          Give a user or a group a level per instance. A global admin is super admin everywhere and
          needs no permission.
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
        instances.map((instance) => <InstancePermissions key={instance.id} instance={instance} />)
      )}
    </div>
  );
}

function InstancePermissions({ instance }: { instance: Connection }) {
  const family = instance.family as FamilyKey | undefined;
  const name = instance.name;
  const queryClient = useQueryClient();
  const { identity } = useAuth();
  const [memberType, setMemberType] = useState<string>("group");
  const [member, setMember] = useState("");
  const [level, setLevel] = useState<string>("read");

  const permissions = useQuery({
    queryKey: ["permissions", family, name],
    queryFn: () => api.permissions.list(family as FamilyKey, name),
    enabled: !!family && !!name,
  });
  const members = useQuery({ queryKey: ["config", "members"], queryFn: () => api.config.members() });

  const knownGroups = useMemo(() => {
    const all = new Set<string>([...(identity?.groups ?? []), ...(members.data?.groups ?? [])]);
    return [...all].sort();
  }, [identity, members.data]);

  const label = (p: { member: string; memberType: string }) =>
    p.memberType === "user" ? (members.data?.users.find((u) => u.subject === p.member)?.label ?? p.member) : p.member;

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: ["permissions", family, name] });
    void queryClient.invalidateQueries({ queryKey: ["me"] });
  }

  const add = useMutation({
    mutationFn: () => api.permissions.set(family as FamilyKey, name, member.trim(), memberType, level),
    onSuccess: () => {
      const added = label({ member: member.trim(), memberType });
      setMember("");
      invalidate();
      toast.success(`${added} can ${level} ${name}`);
    },
    onError: (err) => toast.error(err instanceof Error ? err.message : String(err)),
  });
  const remove = useMutation({
    mutationFn: (member: string) => api.permissions.remove(family as FamilyKey, name, member),
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
          {(permissions.data?.permissions ?? []).length === 0 ? (
            <span className="text-sm text-muted-foreground">no permissions yet</span>
          ) : (
            (permissions.data?.permissions ?? []).map((p) => (
              <Badge key={p.member} variant="secondary" className="gap-1.5 pr-1">
                {p.memberType === "user" && <span className="text-muted-foreground">user</span>}
                {label(p)}
                <span className="text-muted-foreground">
                  · {LEVELS.find((l) => l.value === p.level)?.label ?? p.level}
                </span>
                <button
                  type="button"
                  aria-label={`remove ${p.member}`}
                  className="rounded-sm p-0.5 hover:bg-background/60"
                  onClick={() => remove.mutate(p.member)}
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
            if (member.trim()) add.mutate();
          }}
        >
          <Select
            value={memberType}
            onValueChange={(v) => {
              setMemberType(String(v));
              setMember("");
            }}
          >
            <SelectTrigger className="w-28">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MEMBER_TYPES.map((t) => (
                <SelectItem key={t.value} value={t.value}>
                  {t.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {memberType === "user" ? (
            <Select value={member} onValueChange={(v) => setMember(String(v))}>
              <SelectTrigger className="w-56">
                <SelectValue placeholder="pick a user" />
              </SelectTrigger>
              <SelectContent>
                {(members.data?.users ?? []).map((u) => (
                  <SelectItem key={u.subject} value={u.subject}>
                    {u.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <>
              <Input
                className="w-56"
                placeholder="group name"
                list="known-groups"
                value={member}
                onChange={(e) => setMember(e.target.value)}
              />
              <datalist id="known-groups">
                {knownGroups.map((g) => (
                  <option key={g} value={g} />
                ))}
              </datalist>
            </>
          )}

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
          <Button type="submit" size="sm" disabled={!member.trim() || add.isPending}>
            <Plus /> Grant
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
