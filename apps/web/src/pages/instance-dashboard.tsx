import { useParams } from "react-router-dom";
import { ShieldCheck } from "lucide-react";
import { InstancePermissions } from "@/components/access/instance-permissions";
import { Empty } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { H1, P } from "@/components/ui/typography";
import { useAuth } from "@/app/auth-provider";
import { useInstanceQuery } from "@/hooks/use-instances";

// InstanceDashboardPage is the instance admin's home: who can reach this
// instance and when their access lapses.
export default function InstanceDashboardPage() {
  const { id = "" } = useParams();
  const { isInstanceAdmin } = useAuth();
  const { data: instance, isPending } = useInstanceQuery(id);

  if (isPending || !instance) return <Skeleton className="h-64 w-full" />;
  if (!isInstanceAdmin(instance.name)) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Instance admins only"
          description="Only the super admin or this instance's admin can open its dashboard."
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <H1>{instance.name} · Dashboard</H1>
        <P>Who can reach this instance, and when their access lapses.</P>
      </div>
      <InstancePermissions instance={instance} />
    </div>
  );
}
