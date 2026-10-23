import { useMemo } from "react";
import { KeyRound, Search, ShieldCheck } from "lucide-react";
import { Empty } from "@/components/ui/empty";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import { Skeleton } from "@/components/ui/skeleton";
import { InstancePermissions } from "@/components/access/instance-permissions";
import { useAuth } from "@/app/auth-provider";
import { useAccessMatrixQuery } from "@/hooks/use-access";
import { useUsersUrlParams } from "@/hooks/use-users";
import { H1, P } from "@/components/ui/typography";

// AccessPage is the per-instance access overview: every instance with its whole
// policy and the grant/revoke form inline, so an admin works the whole access
// landscape in one place without needing a grant on each instance. The filter
// lives in the URL, like every other filtered view.
export default function AccessPage() {
  const { isAdmin } = useAuth();
  const { q, setQ } = useUsersUrlParams();
  const matrix = useAccessMatrixQuery(isAdmin);

  const instances = useMemo(() => {
    // the generated schema marks every field optional: an instance without a
    // name is not addressable, so drop it rather than render a ghost card
    const all = (matrix.data?.instances ?? []).flatMap((i) =>
      i.connection ? [{ name: i.connection, driver: i.driver ?? "", family: i.family ?? "" }] : [],
    );
    const needle = q.trim().toLowerCase();
    return needle ? all.filter((i) => i.name.toLowerCase().includes(needle)) : all;
  }, [matrix.data, q]);

  if (!isAdmin) {
    return (
      <div className="mx-auto mt-8 max-w-lg">
        <Empty
          icon={ShieldCheck}
          title="Admin only"
          description="The access overview is reserved for admins."
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div>
        <H1>Access by instance</H1>
        <P>Every instance and who holds which level on it, across all engine families.</P>
      </div>

      <InputGroup className="h-8 w-64">
        <InputGroupAddon>
          <Search />
        </InputGroupAddon>
        <InputGroupInput value={q} onChange={(e) => setQ(e.target.value)} placeholder="Filter by instance" />
      </InputGroup>

      {matrix.isPending ? (
        <Skeleton className="h-32 w-full" />
      ) : instances.length === 0 ? (
        <Empty icon={KeyRound} title="No instances" description="No instance matches this filter." />
      ) : (
        <div className="grid items-start gap-4 xl:grid-cols-2">
          {instances.map((instance) => (
            <InstancePermissions key={instance.name} instance={instance} />
          ))}
        </div>
      )}
    </div>
  );
}
