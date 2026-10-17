import { Link } from "react-router-dom";
import { ShieldCheck, Users as UsersIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/app/auth-provider";
import { useUsersQuery, useUsersUrlParams } from "@/hooks/use-users";
import { formatDateTime } from "@/lib/dates";
import { PAGE_ROUTES } from "@/lib/routes";
import { H1, P } from "@/components/ui/typography";

// UsersPage is the super admin's directory: every account, filterable by a
// search term and status. The filters live in the URL, so a filtered view can
// be shared or reloaded.
export default function UsersPage() {
  const { isSuper } = useAuth();
  const { q, status, setQ, setStatus } = useUsersUrlParams();
  const { data, isPending } = useUsersQuery(isSuper);

  if (!isSuper) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Super admin only"
          description="The user directory is reserved for the super admin."
        />
      </div>
    );
  }

  const users = data?.users ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div>
        <H1>Users</H1>
        <P>
          Every account that can sign in, its role, and when it was last seen.
        </P>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search subject or username"
          className="h-8 w-64"
        />
        <Select value={status} onValueChange={(v) => setStatus(v ?? "all")}>
          <SelectTrigger className="h-8 w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="disabled">Disabled</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isPending ? (
        <Skeleton className="h-24 w-full" />
      ) : users.length === 0 ? (
        <Empty icon={UsersIcon} title="No accounts" description="No account matches this filter." />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Account</TableHead>
                <TableHead>Role</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Groups</TableHead>
                <TableHead>Last login</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.map((u) => (
                <TableRow key={u.subject}>
                  <TableCell>
                    <Link to={PAGE_ROUTES.USER_DETAIL(u.subject)} className="font-medium hover:underline">
                      {u.username || u.subject}
                    </Link>
                    <div className="font-mono text-xs text-muted-foreground">{u.subject}</div>
                  </TableCell>
                  <TableCell>
                    {u.super ? <Badge variant="success">Super</Badge> : <Badge variant="secondary">Member</Badge>}
                  </TableCell>
                  <TableCell>
                    {u.disabled ? (
                      <Badge variant="destructive">Disabled</Badge>
                    ) : (
                      <Badge variant="outline">Active</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {(u.groups ?? []).join(", ") || "—"}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{formatDateTime(u.lastLoginAt)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
