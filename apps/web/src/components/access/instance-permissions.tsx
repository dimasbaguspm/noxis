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

// InstancePermissions is one instance's access policy. The super admin grants
// and revokes; the instance's own admin sees the same list read-only. The
// grant-form choices live in the URL, and levels come from the backend catalog.
export function InstancePermissions({ instance }: { instance: Connection }) {
  const { isSuper } = useAuth();
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
  const members = useMembersQuery(isSuper);
  const levels = useLevelsQuery().data?.levels ?? [];
  const add = useSetPermission(family, name);
  const remove = useRemovePermission(family, name);

  const groups = knownGroups(members.data?.groups);
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
          {!isSuper && " · read-only"}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          {(permissions.data?.permissions ?? []).length === 0 ? (
            <span className="text-sm text-muted-foreground">no permissions yet</span>
          ) : (
            (permissions.data?.permissions ?? []).map((p) => {
              const exp = grantExpiry(p.expiresAt);
              return (
                <Badge key={p.member} variant="secondary" className="gap-1.5 pr-1">
                  {p.memberType === "user" && <span className="text-muted-foreground">user</span>}
                  {memberLabel(p, members.data?.users)}
                  <span className="text-muted-foreground">
                    · {levels.find((l) => l.value === p.level)?.label ?? p.level} · {exp.label}
                  </span>
                  {isSuper && (
                    <button
                      type="button"
                      aria-label={`remove ${p.member}`}
                      className="rounded-sm p-0.5 hover:bg-background/60"
                      onClick={() => remove.mutate(p.member)}
                    >
                      <X className="size-3" />
                    </button>
                  )}
                </Badge>
              );
            })
          )}
        </div>

        {isSuper && (
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
