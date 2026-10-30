import { useInstanceAccess } from "@/hooks/use-instance-access";
import { Check, Inbox, KeyRound, Plus, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import { GrantTable } from "@/components/access/grant-table";
import { knownGroups, memberLabel } from "@/components/access/helpers";
import { Workspace } from "@/components/layout/workspace";
import { useAuth } from "@/app/auth-provider";
import { useDecideRequest, useToApproveQuery } from "@/hooks/use-access-requests";
import {
  useMembersQuery,
  usePermissionsQuery,
  useRemovePermission,
  useSetPermission,
} from "@/hooks/use-access";
import { useLevelsQuery } from "@/hooks/use-config";
import type { Connection } from "@/lib/api";
import { formatDateTime } from "@/lib/dates";
import { familyByKey, familyOfDriver } from "@/lib/families";
import { useUrlState } from "@/hooks/use-url-state";
import { cn } from "@/lib/utils";

const ACCESS_VIEW_DEFAULTS = { view: "granted" };
const MEMBER_TYPES = [
  { value: "group", label: "Group" },
  { value: "user", label: "User" },
] as const;

// InstanceAccess is the instance review surface: grants and pending requests share one 20:80 workspace.
export function InstanceAccess({ instance }: { instance: Connection }) {
  const { isAdmin, isInstanceAdmin } = useAuth();
  const family = familyOfDriver(instance.driver);
  const name = instance.name;
  const [params, patch] = useUrlState(ACCESS_VIEW_DEFAULTS);
  const view = params.view === "requests" ? "requests" : "granted";

  const permissions = usePermissionsQuery(instance.driver, name);
  const members = useMembersQuery(isAdmin);
  const levels = useLevelsQuery().data?.levels ?? [];
  const requests = useToApproveQuery();
  const add = useSetPermission(instance.driver, name);
  const remove = useRemovePermission(instance.driver, name);
  const decide = useDecideRequest();

  const {
    sheet,
    memberType,
    setMemberType,
    member,
    setMember,
    level,
    setLevel,
    note,
    setNote,
    openAdd,
    openGrant,
    openRequest,
    closeSheet,
    submitGrant,
    revoke,
    decideRequest,
  } = useInstanceAccess({ add, remove, decide });
  const grants = permissions.data?.permissions ?? [];
  const pending = (requests.data?.requests ?? []).filter((request) => request.connection === name);
  const groups = knownGroups(members.data?.groups);
  const canGrant = isAdmin;
  const canRevoke = isAdmin || isInstanceAdmin(name);
  const canManage = canGrant || canRevoke;

  return (
    <>
      <div className="flex h-full min-h-0">
        <Workspace
          left={
            <div className="flex min-h-0 flex-1 flex-col gap-1">
              <button
                type="button"
                onClick={() => patch({ view: "granted" })}
                className={cn(
                  "flex items-center gap-2 rounded-md px-2 py-2 text-left text-sm text-muted-foreground hover:bg-accent/60 hover:text-foreground",
                  view === "granted" && "bg-accent text-accent-foreground",
                )}
              >
                <KeyRound className="size-3.5" />
                <span className="flex-1">Granted</span>
                <Badge variant="secondary">{grants.length}</Badge>
              </button>
              <button
                type="button"
                onClick={() => patch({ view: "requests" })}
                className={cn(
                  "flex items-center gap-2 rounded-md px-2 py-2 text-left text-sm text-muted-foreground hover:bg-accent/60 hover:text-foreground",
                  view === "requests" && "bg-accent text-accent-foreground",
                )}
              >
                <Inbox className="size-3.5" />
                <span className="flex-1">Requests</span>
                <Badge variant="secondary">{pending.length}</Badge>
              </button>
              {canGrant && (
                <Button size="sm" variant="outline" className="mt-auto" onClick={openAdd}>
                  <Plus /> Add access
                </Button>
              )}
            </div>
          }
          right={
            view === "granted" ? (
              permissions.isPending ? (
                <Skeleton className="h-48 w-full" />
              ) : grants.length === 0 ? (
                <Empty
                  icon={KeyRound}
                  title="No explicit grants"
                  description="Grant a user or group on this instance to get started."
                />
              ) : (
                <div className="min-h-0 flex-1 overflow-auto">
                  <GrantTable
                    grants={grants}
                    users={members.data?.users}
                    levels={levels}
                    onManage={canManage ? openGrant : undefined}
                  />
                </div>
              )
            ) : requests.isPending ? (
              <Skeleton className="h-48 w-full" />
            ) : pending.length === 0 ? (
              <Empty
                icon={Inbox}
                title="No pending requests"
                description="Access requests waiting for a decision appear here."
              />
            ) : (
              <div className="min-h-0 flex-1 overflow-auto">
                <div className="rounded-md border border-border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Member</TableHead>
                        <TableHead>Level</TableHead>
                        <TableHead>Reason</TableHead>
                        <TableHead>Requested</TableHead>
                        <TableHead className="w-[100px]" />
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {pending.map((request) => (
                        <TableRow key={request.id}>
                          <TableCell className="font-mono text-xs">{request.subject}</TableCell>
                          <TableCell>
                            <Badge variant="secondary">{request.level}</Badge>
                          </TableCell>
                          <TableCell
                            className="max-w-72 truncate text-xs text-muted-foreground"
                            title={request.reason || undefined}
                          >
                            {request.reason || "—"}
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground">
                            {formatDateTime(request.requestedAt)}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button size="xs" variant="ghost" onClick={() => openRequest(request)}>
                              <Inbox /> Review
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            )
          }
        />
      </div>

      <Sheet open={!!sheet} onOpenChange={(open) => (open ? undefined : closeSheet())}>
        <SheetContent className="sm:max-w-md">
          <SheetHeader>
            <SheetTitle>
              {sheet?.kind === "add"
                ? "Add access"
                : sheet?.kind === "grant"
                  ? memberLabel(sheet.permission, members.data?.users)
                  : sheet?.request.subject}
            </SheetTitle>
            <SheetDescription>
              {sheet?.kind === "add"
                ? `Grant a user or group on this ${family ? familyByKey[family].label : "instance"}.`
                : sheet?.kind === "grant"
                  ? "Change the level or revoke this grant."
                  : "Review the request, then approve or reject it."}
            </SheetDescription>
          </SheetHeader>
          <form
            id="access-sheet-form"
            onSubmit={sheet?.kind === "request" ? (event) => event.preventDefault() : submitGrant}
          >
            <SheetBody className="flex flex-col gap-4">
              {sheet?.kind === "request" ? (
                <>
                  <div className="grid gap-2 rounded-md border px-3 py-2 text-sm">
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-muted-foreground">Level</span>
                      <Badge variant="secondary">{sheet.request.level}</Badge>
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-muted-foreground">Requested</span>
                      <span className="text-xs">{formatDateTime(sheet.request.requestedAt)}</span>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {sheet.request.reason || "No reason given."}
                    </div>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="decision-note">Decision note</Label>
                    <Textarea
                      id="decision-note"
                      value={note}
                      onChange={(event) => setNote(event.target.value)}
                      placeholder="Optional"
                    />
                  </div>
                </>
              ) : (
                <>
                  {sheet?.kind === "grant" ? (
                    <div className="rounded-md border px-3 py-2 text-sm">
                      <div className="font-medium">
                        {memberLabel(sheet.permission, members.data?.users)}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {sheet.permission.memberType}
                      </div>
                    </div>
                  ) : (
                    <>
                      <div className="flex flex-col gap-1.5">
                        <Label>Member type</Label>
                        <Select
                          value={memberType}
                          onValueChange={(value) => setMemberType(value ?? "group")}
                        >
                          <SelectTrigger className="w-full">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {MEMBER_TYPES.map((type) => (
                              <SelectItem key={type.value} value={type.value}>
                                {type.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="flex flex-col gap-1.5">
                        <Label>{memberType === "user" ? "User" : "Group"}</Label>
                        {memberType === "user" ? (
                          <Select value={member} onValueChange={(value) => setMember(value ?? "")}>
                            <SelectTrigger className="w-full">
                              <SelectValue placeholder="Pick a user" />
                            </SelectTrigger>
                            <SelectContent>
                              {(members.data?.users ?? []).map((user) => (
                                <SelectItem key={user.subject} value={user.subject ?? ""}>
                                  {user.label}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        ) : (
                          <>
                            <Input
                              placeholder="Group name"
                              list="known-groups"
                              value={member}
                              onChange={(event) => setMember(event.target.value)}
                            />
                            <datalist id="known-groups">
                              {groups.map((group) => (
                                <option key={group} value={group} />
                              ))}
                            </datalist>
                          </>
                        )}
                      </div>
                    </>
                  )}
                  <div className="flex flex-col gap-1.5">
                    <Label>Level</Label>
                    <Select value={level} onValueChange={(value) => setLevel(value ?? "read")}>
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {levels.map((item) => (
                          <SelectItem key={item.value ?? ""} value={item.value ?? ""}>
                            {item.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </>
              )}
            </SheetBody>
            <SheetFooter className="items-center">
              {sheet?.kind === "grant" && canRevoke && (
                <Button
                  type="button"
                  variant="ghost"
                  className="mr-auto text-destructive"
                  onClick={revoke}
                  disabled={remove.isPending}
                >
                  Revoke
                </Button>
              )}
              {sheet?.kind === "request" ? (
                <>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => decideRequest(false)}
                    disabled={decide.isPending}
                  >
                    <X /> Reject
                  </Button>
                  <Button
                    type="button"
                    onClick={() => decideRequest(true)}
                    disabled={decide.isPending}
                  >
                    <Check /> Approve
                  </Button>
                </>
              ) : (
                <Button
                  type="submit"
                  form="access-sheet-form"
                  disabled={!member.trim() || add.isPending}
                >
                  {sheet?.kind === "grant" ? "Save level" : "Grant"}
                </Button>
              )}
            </SheetFooter>
          </form>
        </SheetContent>
      </Sheet>
    </>
  );
}
