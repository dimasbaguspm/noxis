import { useParams } from "react-router-dom";
import { ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { RoleBadge } from "@/components/access/role-badge";
import { useAuth } from "@/app/auth-provider";
import { useLevelsQuery } from "@/hooks/use-config";
import { useUpdateUser, useUserQuery } from "@/hooks/use-users";
import { formatDateTime, grantExpiry } from "@/lib/dates";
import { H1, P } from "@/components/ui/typography";
import type { UserRole } from "@/lib/api";

// UserDetailPage is an admin view of one account: identity, role and status,
// the access it holds directly or through a group, and the activity trail. Only
// the super admin may change the role; an admin may only ban members.
export default function UserDetailPage() {
  const { subject = "" } = useParams();
  const { isAdmin, canBan, canSetRole } = useAuth();
  const { data: user, isPending } = useUserQuery(isAdmin ? subject : "");
  const update = useUpdateUser(subject);
  const levels = useLevelsQuery().data?.levels ?? [];
  const levelLabel = (value?: string) => levels.find((l) => l.value === value)?.label ?? value;

  if (!isAdmin) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty icon={ShieldCheck} title="Admin only" description="Account details are reserved for admins." />
      </div>
    );
  }
  if (isPending || !user) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <H1>{user.email || user.username || user.subject}</H1>
          <P className="font-mono">{user.subject}</P>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <RoleBadge role={user.role} />
          {user.status === "banned" && <Badge variant="destructive">Banned</Badge>}
          {canSetRole(user) && (
            <Select
              value={user.role}
              onValueChange={(v) => v && update.mutate({ role: v as UserRole })}
            >
              <SelectTrigger className="h-8 w-32">
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
              onClick={() => update.mutate({ status: user.status === "banned" ? "active" : "banned" })}
            >
              {user.status === "banned" ? "Reinstate" : "Ban"}
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Identity</CardTitle>
          <CardDescription>
            What the directory knows about this account. An OIDC login with this email signs in as this account.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-muted-foreground">Email</dt>
              <dd>{user.email || "—"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Status</dt>
              <dd>{user.status === "banned" ? "banned" : "active"}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Last login</dt>
              <dd>{formatDateTime(user.lastLoginAt)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Created</dt>
              <dd>{formatDateTime(user.createdAt)}</dd>
            </div>
            <div className="sm:col-span-2">
              <dt className="text-muted-foreground">Groups</dt>
              <dd className="mt-1 flex flex-wrap gap-1.5">
                {(user.groups ?? []).length === 0 ? (
                  <span className="text-muted-foreground">none</span>
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
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Access</CardTitle>
          <CardDescription>Grants held directly or through a group, and when they lapse.</CardDescription>
        </CardHeader>
        <CardContent>
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
                    no access
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
                      {exp.expired ? <Badge variant="destructive">{exp.label}</Badge> : exp.label}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{grant.createdBy || "system"}</TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Activity</CardTitle>
          <CardDescription>Access events about this account, newest first.</CardDescription>
        </CardHeader>
        <CardContent>
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
                    no activity
                  </TableCell>
                </TableRow>
              )}
              {(user.events ?? []).map((event) => (
                <TableRow key={event.id}>
                  <TableCell className="text-xs text-muted-foreground">{formatDateTime(event.at)}</TableCell>
                  <TableCell className="text-sm">{event.action}</TableCell>
                  <TableCell className="text-sm">{event.connection || "—"}</TableCell>
                  <TableCell className="text-sm">{levelLabel(event.level) || "—"}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{event.actor}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
