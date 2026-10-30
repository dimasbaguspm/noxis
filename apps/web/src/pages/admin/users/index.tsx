import { Link } from "react-router-dom";
import { Search, ShieldCheck, Users as UsersIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Empty } from "@/components/ui/empty";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
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
import { useAuth } from "@/app/auth-provider";
import { RoleBadge } from "@/components/access/role-badge";
import { Workspace } from "@/components/layout/workspace";
import { useUsersQuery, useUsersUrlParams } from "@/hooks/use-users";
import { formatDateTime } from "@/lib/dates";
import { PAGE_ROUTES } from "@/constants/routes";

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
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="user-search">Search</Label>
              <InputGroup>
                <InputGroupAddon>
                  <Search />
                </InputGroupAddon>
                <InputGroupInput
                  id="user-search"
                  value={q}
                  onChange={(e) => setQ(e.target.value)}
                  placeholder="Email or subject"
                />
              </InputGroup>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Status</Label>
              <Select value={status} onValueChange={(v) => setStatus(v ?? "all")}>
                <SelectTrigger className="w-full">
                  <SelectValue>
                    {(value) =>
                      value === "all" ? "All accounts" : value === "active" ? "Active" : "Banned"
                    }
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All accounts</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="banned">Banned</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        }
        right={
          <div className="min-h-0 flex-1 overflow-auto">
            {isPending ? (
              <Skeleton className="h-64 w-full" />
            ) : users.length === 0 ? (
              <Empty
                icon={UsersIcon}
                title="No accounts"
                description="No account matches this filter."
              />
            ) : (
              <div className="rounded-md border">
                <Table>
                  <TableHeader className="sticky top-0 bg-card">
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
                          <Link
                            to={PAGE_ROUTES.USER_DETAIL(u.subject)}
                            className="font-medium hover:underline"
                          >
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
                        <TableCell className="text-xs text-muted-foreground">
                          {formatDateTime(u.lastLoginAt)}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        }
      />
    </div>
  );
}
