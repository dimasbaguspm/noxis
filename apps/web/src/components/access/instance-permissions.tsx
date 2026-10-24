import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Plus, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAuth } from "@/app/auth-provider";
import { knownGroups, memberLabel } from "@/components/access/helpers";
import {
  useMembersQuery,
  usePermissionsQuery,
  useRemovePermission,
  useSetPermission,
} from "@/hooks/use-access";
import { useLevelsQuery } from "@/hooks/use-config";
import type { Connection } from "@/lib/api";
import { grantExpiry } from "@/lib/dates";
import { familyByKey, type FamilyKey } from "@/lib/families";

const MEMBER_TYPES = [
  { value: "group", label: "Group" },
  { value: "user", label: "User" },
] as const;

// InstancePermissions is one instance's access policy: everyone who can reach it, at what level and until when.
export function InstancePermissions({
  instance,
  readOnly = false,
}: {
  instance: Pick<Connection, "name" | "driver" | "family">
  readOnly?: boolean
}) {
  const { isAdmin, isInstanceAdmin } = useAuth();
  const family = instance.family as FamilyKey | undefined;
  const name = instance.name;
  const [params, setParams] = useSearchParams();
  const memberType = params.get("memberType") ?? "group";
  const level = params.get("level") ?? "read";
  const [member, setMember] = useState("");

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next, { replace: true });
  };

  const permissions = usePermissionsQuery(family, name);
  const members = useMembersQuery(isAdmin);
  const levels = useLevelsQuery().data?.levels ?? [];
  const add = useSetPermission(family, name);
  const remove = useRemovePermission(family, name);

  const groups = knownGroups(members.data?.groups);
  const meta = family ? familyByKey[family] : undefined;
  const Icon = meta?.icon;
  const canGrant = isAdmin && !readOnly;
  const canRevoke = (isAdmin || isInstanceAdmin(name)) && !readOnly;
  const granted = new Set((permissions.data?.permissions ?? []).map((p) => p.member));
  // supers hold no grants: their access is implicit, so list only those without an explicit row
  const implicitSupers = (members.data?.users ?? []).filter(
    (u) => u.role === "super" && u.subject && !granted.has(u.subject),
  );

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
        {(permissions.data?.permissions ?? []).length === 0 ? (
          <span className="text-sm text-muted-foreground">no permissions yet</span>
        ) : (
          <ul className="flex flex-col gap-2">
            {(permissions.data?.permissions ?? []).map((p) => {
              const exp = grantExpiry(p.expiresAt);
              const level = levels.find((l) => l.value === p.level)?.label ?? p.level;
              const account = members.data?.users?.find((u) => u.subject === p.member);
              return (
                <li key={p.member} className="flex items-center gap-3 rounded-md border border-border px-3 py-2">
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{memberLabel(p, members.data?.users)}</div>
                    <div className="truncate text-xs text-muted-foreground">
                      {p.memberType} · {level} · {exp.label}
                      {account?.role ? ` · ${account.role}` : ""}
                    </div>
                  </div>
                  {canRevoke && (
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-label={`revoke ${p.member}`}
                      disabled={remove.isPending}
                      onClick={() => remove.mutate(p.member)}
                    >
                      <X /> Revoke
                    </Button>
                  )}
                </li>
              );
            })}
          </ul>
        )}

        {implicitSupers.length > 0 && (
          <div className="rounded-md border border-dashed px-3 py-2">
            <p className="text-xs font-medium text-muted-foreground">Superadmins · implicit access</p>
            <div className="mt-1 flex flex-wrap gap-1.5">
              {implicitSupers.map((u) => (
                <Badge key={u.subject} variant="outline">
                  {u.label ?? u.subject}
                </Badge>
              ))}
            </div>
          </div>
        )}

        {canGrant && (
          <form
            className="flex flex-wrap items-center gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              if (member.trim()) {
                add.mutate(
                  { member: member.trim(), memberType, level },
                  { onSuccess: () => setMember("") },
                );
              }
            }}
          >
            <Select value={memberType} onValueChange={(v) => setParam("memberType", v ?? "group")}>
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
              <Select value={member} onValueChange={(v) => setMember(v ?? "")}>
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
                  {groups.map((g) => (
                    <option key={g} value={g} />
                  ))}
                </datalist>
              </>
            )}

            <Select value={level} onValueChange={(v) => setParam("level", v ?? "read")}>
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {levels.map((l) => (
                  <SelectItem key={l.value ?? ""} value={l.value ?? ""}>
                    {l.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button type="submit" size="sm" disabled={!member.trim() || add.isPending}>
              <Plus /> Grant
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
