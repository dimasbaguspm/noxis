import { useState } from "react";
import { Plus, Settings2, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/app/auth-provider";
import { knownGroups, memberLabel } from "@/components/access/helpers";
import {
  useMembersQuery,
  usePermissionsQuery,
  useRemovePermission,
  useSetPermission,
} from "@/hooks/use-access";
import { useLevelsQuery } from "@/hooks/use-config";
import type { Connection, Permission } from "@/lib/api";
import { grantExpiry } from "@/lib/dates";
import { familyByKey, type FamilyKey } from "@/lib/families";

const MEMBER_TYPES = [
  { value: "group", label: "Group" },
  { value: "user", label: "User" },
] as const;

// InstancePermissions is one instance's access policy: a direct grant table plus a sheet to add, change or revoke a member.
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

  const permissions = usePermissionsQuery(family, name);
  const members = useMembersQuery(isAdmin);
  const levels = useLevelsQuery().data?.levels ?? [];
  const add = useSetPermission(family, name);
  const remove = useRemovePermission(family, name);

  const [sheetOpen, setSheetOpen] = useState(false);
  const [editing, setEditing] = useState<Permission | null>(null);
  const [memberType, setMemberType] = useState<string>("group");
  const [member, setMember] = useState("");
  const [level, setLevel] = useState("read");

  const grants = permissions.data?.permissions ?? [];
  const groups = knownGroups(members.data?.groups);
  const meta = family ? familyByKey[family] : undefined;
  const canGrant = isAdmin && !readOnly;
  const canRevoke = (isAdmin || isInstanceAdmin(name)) && !readOnly;
  const canManage = canGrant || canRevoke;
  const granted = new Set(grants.map((p) => p.member));
  // supers hold no grants: their access is implicit, so list only those without an explicit row
  const implicitSupers = (members.data?.users ?? []).filter(
    (u) => u.role === "super" && u.subject && !granted.has(u.subject),
  );

  function openAdd() {
    setEditing(null);
    setMemberType("group");
    setMember("");
    setLevel("read");
    setSheetOpen(true);
  }

  function openManage(permission: Permission) {
    setEditing(permission);
    setMemberType(permission.memberType ?? "group");
    setMember(permission.member);
    setLevel(permission.level ?? "read");
    setSheetOpen(true);
  }

  function closeSheet() {
    setSheetOpen(false);
    setEditing(null);
  }

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!member.trim()) return;
    add.mutate(
      { member: member.trim(), memberType, level },
      { onSuccess: closeSheet },
    );
  }

  function revoke() {
    if (!editing) return;
    remove.mutate(editing.member, { onSuccess: closeSheet });
  }

  return (
    <>
      <Card>
        <CardHeader className="gap-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <CardTitle className="text-base">Access</CardTitle>
              <CardDescription>{grants.length} explicit grant(s) on this instance.</CardDescription>
            </div>
            {canGrant && (
              <Button size="sm" onClick={openAdd}>
                <Plus /> Add
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {grants.length === 0 ? (
            <p className="text-sm text-muted-foreground">No explicit grants yet.</p>
          ) : (
            <div className="rounded-md border border-border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Member</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Level</TableHead>
                    <TableHead>Expires</TableHead>
                    {canManage && <TableHead className="w-[110px]" />}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {grants.map((p) => {
                    const exp = grantExpiry(p.expiresAt);
                    const levelLabel = levels.find((l) => l.value === p.level)?.label ?? p.level;
                    return (
                      <TableRow key={p.member}>
                        <TableCell className="text-sm font-medium">{memberLabel(p, members.data?.users)}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{p.memberType}</Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary">{levelLabel}</Badge>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">{exp.label}</TableCell>
                        {canManage && (
                          <TableCell className="text-right">
                            <Button variant="ghost" size="xs" onClick={() => openManage(p)}>
                              <Settings2 /> Manage
                            </Button>
                          </TableCell>
                        )}
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
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
        </CardContent>
      </Card>

      <Sheet open={sheetOpen} onOpenChange={(open) => (open ? setSheetOpen(true) : closeSheet())}>
        <SheetContent className="sm:max-w-md">
          <SheetHeader>
            <SheetTitle>{editing ? memberLabel(editing, members.data?.users) : "Add access"}</SheetTitle>
            <SheetDescription>
              {editing ? "Change the level or revoke this grant." : `Grant a user or group on this ${meta?.label ?? "instance"}.`}
            </SheetDescription>
          </SheetHeader>
          <form id="permission-form" onSubmit={submit}>
            <SheetBody className="flex flex-col gap-4">
              {editing ? (
                <div className="rounded-md border px-3 py-2 text-sm">
                  <div className="font-medium">{memberLabel(editing, members.data?.users)}</div>
                  <div className="text-xs text-muted-foreground">{editing.memberType}</div>
                </div>
              ) : (
                <>
                  <div className="flex flex-col gap-1.5">
                    <Label>Member type</Label>
                    <Select value={memberType} onValueChange={(v) => setMemberType(v ?? "group")}>
                      <SelectTrigger className="w-full">
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
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label>{memberType === "user" ? "User" : "Group"}</Label>
                    {memberType === "user" ? (
                      <Select value={member} onValueChange={(v) => setMember(v ?? "")}>
                        <SelectTrigger className="w-full">
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
                  </div>
                </>
              )}
              <div className="flex flex-col gap-1.5">
                <Label>Level</Label>
                <Select value={level} onValueChange={(v) => setLevel(v ?? "read")}>
                  <SelectTrigger className="w-full">
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
              </div>
            </SheetBody>
            <SheetFooter className="items-center">
              {editing && canRevoke && (
                <Button type="button" variant="ghost" className="mr-auto text-destructive" onClick={revoke} disabled={remove.isPending}>
                  <X /> Revoke
                </Button>
              )}
              <Button type="submit" form="permission-form" disabled={!member.trim() || add.isPending}>
                {editing ? "Save level" : "Grant"}
              </Button>
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>
    </>
  );
}
