import { Settings2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { memberLabel } from "@/components/access/helpers";
import { grantExpiry } from "@/lib/dates";
import type { AccessLevel, MemberUser, Permission } from "@/lib/api";

// GrantTable is the shared read surface for a connection's direct grants.
export function GrantTable({
  grants,
  users,
  levels,
  onManage,
}: {
  grants: Permission[];
  users?: MemberUser[];
  levels: AccessLevel[];
  onManage?: (permission: Permission) => void;
}) {
  const levelLabel = (value?: string) => levels.find((level) => level.value === value)?.label ?? value;

  return (
    <div className="rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Member</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Level</TableHead>
            <TableHead>Expires</TableHead>
            {onManage && <TableHead className="w-[110px]" />}
          </TableRow>
        </TableHeader>
        <TableBody>
          {grants.map((permission) => {
            const expiry = grantExpiry(permission.expiresAt);
            return (
              <TableRow key={permission.member}>
                <TableCell className="text-sm font-medium">{memberLabel(permission, users)}</TableCell>
                <TableCell>
                  <Badge variant="outline">{permission.memberType}</Badge>
                </TableCell>
                <TableCell>
                  <Badge variant="secondary">{levelLabel(permission.level)}</Badge>
                </TableCell>
                <TableCell className="text-xs text-muted-foreground">{expiry.label}</TableCell>
                {onManage && (
                  <TableCell className="text-right">
                    <Button variant="ghost" size="xs" onClick={() => onManage(permission)}>
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
  );
}
