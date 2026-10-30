import { Empty } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { KeyRound } from "lucide-react";
import { GrantTable } from "@/components/access/grant-table";
import { useMembersQuery, usePermissionsQuery } from "@/hooks/use-access";
import { useDriversQuery, useLevelsQuery } from "@/hooks/use-config";
import type { Connection } from "@/lib/api";
import { driverLabel } from "@/lib/console-nav";

// InstancePermissions is the read-only grant view used by the global Access page. The grants live under the connection's driver path; the driver label is only the wording of the empty state.
export function InstancePermissions({
  instance,
}: {
  instance: Pick<Connection, "name" | "driver">;
}) {
  const permissions = usePermissionsQuery(instance.driver, instance.name);
  const members = useMembersQuery(true);
  const levels = useLevelsQuery().data?.levels ?? [];
  const grants = permissions.data?.permissions ?? [];
  const { data: drivers } = useDriversQuery();
  const label = driverLabel(drivers, instance.driver);

  if (permissions.isPending) return <Skeleton className="h-48 w-full" />;
  if (grants.length === 0) {
    return (
      <Empty
        icon={KeyRound}
        title="No explicit grants"
        description={`${label || "This instance"} has no direct user or group grant.`}
      />
    );
  }

  return <GrantTable grants={grants} users={members.data?.users} levels={levels} />;
}
