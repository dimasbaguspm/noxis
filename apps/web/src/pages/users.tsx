import { Link } from "react-router-dom";
import { Search, ShieldCheck, Users as UsersIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Empty } from "@/components/ui/empty";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAuth } from "@/app/auth-provider";
import { RoleBadge } from "@/components/access/role-badge";
import { useUsersQuery, useUsersUrlParams } from "@/hooks/use-users";
import { formatDateTime } from "@/lib/dates";
import { PAGE_ROUTES } from "@/lib/routes";
import { H1, P } from "@/components/ui/typography";

// UsersPage lists accounts with shareable URL-backed search and status filters.
export default function UsersPage() {
  const { isAdmin } = useAuth();
  const { q, status, setQ, setStatus } = useUsersUrlParams();
  const { data, isPending } = useUsersQuery(isAdmin);

  if (!isAdmin) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Admin only"
          description="The user directory is reserved for admins."
        />
      </div>
    );
  }

  const users = data?.users ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div>
        <H1>Users</H1>
        <P>Every account that can sign in, its role, and when it was last seen.</P>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <InputGroup className="h-8 w-64">
          <InputGroupAddon>
            <Search />
          </InputGroupAddon>
          <InputGroupInput value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search email or subject" />
        </InputGroup>
        <Select value={status} onValueChange={(v) => setStatus(v ?? "all")}>
          <SelectTrigger className="h-8 w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="banned">Banned</SelectItem>
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
                      {u.email || u.username || u.subject}
                    </Link>
                    <div className="font-mono text-xs text-muted-foreground">{u.subject}</div>
                  </TableCell>
                  <TableCell>
                    <RoleBadge role={u.role} />
                  </TableCell>
                  <TableCell>
                    {u.status === "banned" ? (
                      <Badge variant="destructive">Banned</Badge>
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
