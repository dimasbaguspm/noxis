import { useEffect, useMemo } from "react";
import { KeyRound, Search, ShieldCheck } from "lucide-react";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { InstancePermissions } from "@/components/access/instance-permissions";
import { useAuth } from "@/app/auth-provider";
import { useAccessMatrixQuery } from "@/hooks/use-access";
import { useDriversQuery } from "@/hooks/use-config";
import { useUrlState } from "@/hooks/use-url-state";
import { capabilitiesOf, capabilityIcon } from "@/lib/console-nav";
import { cn } from "@/lib/utils";
import { Workspace } from "@/components/layout/workspace";

const ACCESS_DEFAULTS = { q: "", instance: "" };

// AccessPage keeps the instance list in view while one policy is edited: a resizable list on the left, the selected policy on the right.
export default function AccessPage() {
  const { isAdmin } = useAuth();
  const [state, patch] = useUrlState(ACCESS_DEFAULTS);
  const matrix = useAccessMatrixQuery(isAdmin);
  const { data: drivers } = useDriversQuery();

  const instances = useMemo(() => {
    // the generated schema marks every field optional: an instance without a name is not addressable, so drop it rather than render a ghost row
    const all = (matrix.data?.instances ?? []).flatMap((i) =>
      i.connection
        ? [
            {
              id: i.id ?? "",
              name: i.connection,
              driver: i.driver ?? "",
              grants: i.permissions?.length ?? 0,
            },
          ]
        : [],
    );
    const needle = state.q.trim().toLowerCase();
    return needle ? all.filter((i) => i.name.toLowerCase().includes(needle)) : all;
  }, [matrix.data, state.q]);

  const selected =
    instances.find((i) => i.id === state.instance) ??
    instances.find((i) => i.name === state.instance) ??
    instances[0] ??
    null;

  useEffect(() => {
    if (selected && selected.id !== state.instance) patch({ instance: selected.id });
  }, [selected, state.instance, patch]);

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
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-3">
            <div className="relative">
              <Search className="pointer-events-none absolute top-2 left-2 size-3.5 text-muted-foreground" />
              <Input
                value={state.q}
                onChange={(e) => patch({ q: e.target.value })}
                placeholder="Filter by instance"
                className="h-8 pl-7"
                aria-label="Filter by instance"
              />
            </div>
            <div className="min-h-0 flex-1 overflow-auto">
              {matrix.isPending ? (
                <Skeleton className="h-48 w-full" />
              ) : instances.length === 0 ? (
                <Empty
                  icon={KeyRound}
                  title="No instances"
                  description="No instance matches this view."
                />
              ) : (
                <div className="grid gap-0.5">
                  {instances.map((instance) => {
                    const Icon = capabilityIcon(capabilitiesOf(drivers, instance.driver));
                    return (
                      <button
                        key={instance.id || instance.name}
                        type="button"
                        onClick={() => patch({ instance: instance.id || instance.name })}
                        className={cn(
                          "flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-sm hover:bg-accent/60",
                          instance.id === selected?.id && "bg-accent text-accent-foreground",
                        )}
                      >
                        <Icon className="size-3.5 shrink-0 text-muted-foreground" />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate">{instance.name}</span>
                          <span className="block truncate text-[10px] text-muted-foreground">
                            {instance.driver} · {instance.grants} grant(s)
                          </span>
                        </span>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        }
        right={
          matrix.isPending ? (
            <Skeleton className="h-64 w-full" />
          ) : selected ? (
            <div className="min-h-0 flex-1 overflow-auto">
              <InstancePermissions instance={selected} />
            </div>
          ) : (
            <Empty
              icon={KeyRound}
              title="No instance selected"
              description="Pick an instance on the left."
            />
          )
        }
      />
    </div>
  );
}
