import { useParams } from "react-router-dom";
import { ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RoleBadge } from "@/components/access/role-badge";
import { useAuth } from "@/app/auth-provider";
import { useLevelsQuery } from "@/hooks/use-config";
import { useUpdateUser, useUserQuery } from "@/hooks/use-users";
import { formatDateTime, grantExpiry } from "@/lib/dates";
import type { UserRole } from "@/lib/api";
import { Workspace } from "@/components/layout/workspace";
import { PAGE_ROUTES } from "@/constants/routes";
import { useCrumbLabel } from "@/app/crumb-provider";

// UserDetailPage shows account, grants, and activity; only super changes global roles, admins may ban members.
export default function UserDetailPage() {
  const { subject = "" } = useParams();
  const { isAdmin, canBan, canSetRole } = useAuth();
  const { data: user, isPending } = useUserQuery(isAdmin ? subject : "");
  const update = useUpdateUser(subject);
  // the trail names the account instead of its subject once the record lands
  useCrumbLabel(PAGE_ROUTES.USER_DETAIL(subject), user?.email || user?.username);
  const levels = useLevelsQuery().data?.levels ?? [];
  const levelLabel = (value?: string) => levels.find((l) => l.value === value)?.label ?? value;

  if (!isAdmin) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Admin only"
          description="Account details are reserved for admins."
        />
      </div>
    );
  }
  if (isPending || !user) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-wrap items-center gap-2">
              <RoleBadge role={user.role} />
              {user.status === "banned" && <Badge variant="destructive">Banned</Badge>}
            </div>
            <dl className="grid gap-x-4 gap-y-3 text-sm">
              <div>
                <dt className="text-muted-foreground">Email</dt>
                <dd className="break-words">{user.email || "—"}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Subject</dt>
                <dd className="font-mono text-xs break-all">{user.subject}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Last login</dt>
                <dd>{formatDateTime(user.lastLoginAt)}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Created</dt>
                <dd>{formatDateTime(user.createdAt)}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground">Groups</dt>
                <dd className="mt-1 flex flex-wrap gap-1.5">
                  {(user.groups ?? []).length === 0 ? (
                    <span className="text-muted-foreground">None</span>
                  ) : (
                    (user.groups ?? []).map((group) => (
                      <Badge key={group} variant="outline">
                        {group}
                      </Badge>
                    ))
                  )}
                </dd>
              </div>
            </dl>
            {(canSetRole(user) || canBan(user)) && (
              <div className="mt-auto flex flex-col gap-2 border-t border-border pt-3">
                {canSetRole(user) && (
                  <Select
                    value={user.role}
                    onValueChange={(v) => v && update.mutate({ role: v as UserRole })}
                  >
                    <SelectTrigger className="h-8 w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="member">Member</SelectItem>
                      <SelectItem value="admin">Admin</SelectItem>
                    </SelectContent>
                  </Select>
                )}
                {canBan(user) && (
                  <Button
                    size="sm"
                    variant={user.status === "banned" ? "outline" : "destructive"}
                    disabled={update.isPending}
                    onClick={() =>
                      update.mutate({ status: user.status === "banned" ? "active" : "banned" })
                    }
                  >
                    {user.status === "banned" ? "Reinstate" : "Ban"}
                  </Button>
                )}
              </div>
            )}
          </div>
        }
        right={
          <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-auto">
            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Access
              </h3>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Connection</TableHead>
                    <TableHead>Level</TableHead>
                    <TableHead>Granted to</TableHead>
                    <TableHead>Expiry</TableHead>
                    <TableHead>Granted by</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(user.access ?? []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="text-muted-foreground">
                        No access
                      </TableCell>
                    </TableRow>
                  )}
                  {(user.access ?? []).map((grant) => {
                    const exp = grantExpiry(grant.expiresAt);
                    return (
                      <TableRow key={`${grant.connection}-${grant.member}`}>
                        <TableCell className="text-sm">{grant.connection}</TableCell>
                        <TableCell>
                          <Badge variant="secondary">{levelLabel(grant.level)}</Badge>
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {grant.memberType === "group" ? `group ${grant.member}` : "direct"}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {exp.expired ? (
                            <Badge variant="destructive">{exp.label}</Badge>
                          ) : (
                            exp.label
                          )}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {grant.createdBy || "system"}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Activity
              </h3>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>When</TableHead>
                    <TableHead>Action</TableHead>
                    <TableHead>Connection</TableHead>
                    <TableHead>Level</TableHead>
                    <TableHead>By</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(user.events ?? []).length === 0 && (
                    <TableRow>
                      <TableCell colSpan={5} className="text-muted-foreground">
                        No activity
                      </TableCell>
                    </TableRow>
                  )}
                  {(user.events ?? []).map((event) => (
                    <TableRow key={event.id}>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatDateTime(event.at)}
                      </TableCell>
                      <TableCell className="text-sm">{event.action}</TableCell>
                      <TableCell className="text-sm">{event.connection || "—"}</TableCell>
                      <TableCell className="text-sm">{levelLabel(event.level) || "—"}</TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {event.actor}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </section>
          </div>
        }
      />
    </div>
  );
}
