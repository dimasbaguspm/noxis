import { useParams } from "react-router-dom";
import { ShieldCheck } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/app/auth-provider";
import { useLevelsQuery } from "@/hooks/use-config";
import { useSetUserDisabled, useUserQuery } from "@/hooks/use-users";
import { formatDateTime, grantExpiry } from "@/lib/dates";

// UserDetailPage is the super admin's view of one account: identity, the access
// it holds directly or through a group, and the activity trail.
export default function UserDetailPage() {
  const { subject = "" } = useParams();
  const { isSuper } = useAuth();
  const { data: user, isPending } = useUserQuery(isSuper ? subject : "");
  const toggle = useSetUserDisabled(subject);
  const levels = useLevelsQuery().data?.levels ?? [];
  const levelLabel = (value?: string) => levels.find((l) => l.value === value)?.label ?? value;

  if (!isSuper) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Super admin only"
          description="Account details are reserved for the super admin."
        />
      </div>
    );
  }
  if (isPending || !user) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-base font-medium">{user.username || user.subject}</h1>
          <p className="font-mono text-xs text-muted-foreground">{user.subject}</p>
        </div>
        <div className="flex items-center gap-2">
          {user.super ? <Badge variant="success">Super</Badge> : <Badge variant="secondary">Member</Badge>}
          {user.disabled && <Badge variant="destructive">Disabled</Badge>}
          {!user.super && (
            <Button
              size="sm"
              variant={user.disabled ? "outline" : "destructive"}
              disabled={toggle.isPending}
              onClick={() => toggle.mutate(!user.disabled)}
            >
              {user.disabled ? "Enable" : "Disable"}
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Identity</CardTitle>
          <CardDescription>What the directory knows about this account.</CardDescription>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-x-6 gap-y-3 text-sm sm:grid-cols-2">
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
