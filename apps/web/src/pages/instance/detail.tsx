import { Link } from "react-router-dom";
import { Database } from "lucide-react";
import { Empty } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import { P } from "@/components/ui/typography";
import { Workspace } from "@/components/layout/workspace";
import { FAMILY } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";
import { useInstanceContext } from "./layout";

export default function InstanceDetailTab() {
  const { instance, family, canRead, tables, schemaPending, schemas } = useInstanceContext();
  const fileDriver = instance.driver === "sqlite";
  const databaseLabel = fileDriver
    ? "File path"
    : family === FAMILY.SQL
      ? "Database"
      : family === FAMILY.MONGODB
        ? "Database (auth)"
        : "DB index";

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <dl className="grid grid-cols-[6rem_1fr] gap-y-2 text-sm">
              {!fileDriver && (
                <>
                  <dt className="text-muted-foreground">Endpoint</dt>
                  <dd className="font-mono text-xs">
                    {instance.host}:{instance.port}
                  </dd>
                </>
              )}
              <dt className="text-muted-foreground">{databaseLabel}</dt>
              <dd>{instance.database}</dd>
              {instance.user && (
                <>
                  <dt className="text-muted-foreground">User</dt>
                  <dd>{instance.user}</dd>
                </>
              )}
              <dt className="text-muted-foreground">Registered</dt>
              <dd className="text-xs">
                {instance.createdAt ? new Date(instance.createdAt * 1000).toLocaleString() : "—"}
                {instance.createdBy ? ` · by ${instance.createdBy}` : ""}
              </dd>
            </dl>
            <P className="border-t border-border pt-3">
              Host, database and user are fixed once registered. Only the password can be rotated.
            </P>
          </div>
        }
        right={
          family !== FAMILY.SQL || !canRead ? (
            <Empty icon={Database} title="No schema" description="This instance does not expose SQL tables." />
          ) : schemaPending ? (
            <Skeleton className="h-48 w-full" />
          ) : schemas.length === 0 ? (
            <Empty icon={Database} title="No schemas" description="This instance has no tables yet." />
          ) : (
            <div className="min-h-0 flex-1 overflow-auto">
              <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
                {schemas.map((s) => {
                  const count = tables.filter((t) => t.schema === s).length;
                  return (
                    <Link
                      key={s}
                      to={`${PAGE_ROUTES.INSTANCE_TAB(instance.id ?? "", "tables")}?schema=${encodeURIComponent(s)}`}
                      className="rounded-lg border p-3 text-left hover:bg-accent/50"
                    >
                      <div className="truncate font-mono text-xs">{s}</div>
                      <div className="mt-1 text-lg font-medium tabular-nums">{count}</div>
                      <div className="text-[10px] text-muted-foreground">table(s)</div>
                    </Link>
                  );
                })}
              </div>
            </div>
          )
        }
      />
    </div>
  );
}
