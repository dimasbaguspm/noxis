import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { RequestAccessDialog } from "@/components/access/request-access-dialog";
import { useAuth } from "@/app/auth-provider";
import { useMyAccessUrlParams, useMyRequestsQuery } from "@/hooks/use-access-requests";
import { useLevelsQuery } from "@/hooks/use-config";
import { formatDateTime, grantExpiry } from "@/lib/dates";

// MyAccessPage lists the grants the caller holds (and when they lapse), the requests they filed, and the way to ask for more.
export default function MyAccessPage() {
  const { identity } = useAuth();
  const { status, setStatus } = useMyAccessUrlParams();
  const requests = useMyRequestsQuery();
  const levels = useLevelsQuery().data?.levels ?? [];
  const levelLabel = (value?: string) => levels.find((l) => l.value === value)?.label ?? value;

  const grants = Object.entries(identity?.connections ?? {})
    .map(([connection, grant]) => ({ connection, ...grant, ...grantExpiry(grant.expiresAt) }))
    .filter((g) => (status === "all" ? true : status === "expired" ? g.expired : !g.expired));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-end gap-3">
        <div className="flex items-center gap-2">
          <Select value={status} onValueChange={(v) => setStatus(v ?? "all")}>
            <SelectTrigger className="h-8 w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="expired">Expired</SelectItem>
            </SelectContent>
          </Select>
          <RequestAccessDialog />
        </div>
      </div>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Granted</CardTitle>
          <CardDescription>Every grant expires; only the super admin never does.</CardDescription>
        </CardHeader>
        <CardContent>
          {grants.length === 0 ? (
            <Empty title="No access" description="Ask for access to an instance to get started." />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Connection</TableHead>
                  <TableHead>Level</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {grants.map((g) => (
                  <TableRow key={g.connection}>
                    <TableCell className="text-sm">{g.connection}</TableCell>
                    <TableCell>
                      <Badge variant="secondary">{levelLabel(g.level)}</Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {g.expired ? <Badge variant="destructive">{g.label}</Badge> : g.label}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="gap-1">
          <CardTitle className="text-base">Requests</CardTitle>
          <CardDescription>What you asked for, and what was decided.</CardDescription>
        </CardHeader>
        <CardContent>
          {(requests.data?.requests ?? []).length === 0 ? (
            <Empty title="No requests" description="You have not asked for access yet." />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Connection</TableHead>
                  <TableHead>Level</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Requested</TableHead>
                  <TableHead>Decided by</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(requests.data?.requests ?? []).map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="text-sm">{r.connection}</TableCell>
                    <TableCell>
                      <Badge variant="secondary">{levelLabel(r.level)}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          r.status === "approved"
                            ? "success"
                            : r.status === "rejected"
                              ? "destructive"
                              : "outline"
                        }
                      >
                        {r.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDateTime(r.requestedAt)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {r.decidedBy || "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
