import { Empty } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { KeyRound } from "lucide-react";
import { GrantTable } from "@/components/access/grant-table";
import { useMembersQuery, usePermissionsQuery } from "@/hooks/use-access";
import { useLevelsQuery } from "@/hooks/use-config";
import type { Connection } from "@/lib/api";
import { familyByKey, type FamilyKey } from "@/lib/families";

// InstancePermissions is the read-only grant view used by the global Access page.
export function InstancePermissions({
  instance,
}: {
  instance: Pick<Connection, "name" | "driver" | "family">;
}) {
  const family = instance.family as FamilyKey | undefined;
  const permissions = usePermissionsQuery(family, instance.name);
  const members = useMembersQuery(true);
  const levels = useLevelsQuery().data?.levels ?? [];
  const grants = permissions.data?.permissions ?? [];

  if (permissions.isPending) return <Skeleton className="h-48 w-full" />;
  if (grants.length === 0) {
    return (
      <Empty
        icon={KeyRound}
        title="No explicit grants"
        description={`${familyByKey[family ?? "sql"]?.label ?? "This instance"} has no direct user or group grant.`}
      />
    );
  }

  return <GrantTable grants={grants} users={members.data?.users} levels={levels} />;
}
