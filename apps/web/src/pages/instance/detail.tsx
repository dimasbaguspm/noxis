import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { P } from "@/components/ui/typography";
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
    <div className="flex flex-col gap-4">
      <Card>
        <CardContent className="pt-6">
          <dl className="grid grid-cols-[8rem_1fr] gap-y-2 text-sm sm:grid-cols-[10rem_1fr]">
            <dt className="text-muted-foreground">Driver</dt>
            <dd>
              <Badge variant="secondary">{instance.driver}</Badge>
            </dd>
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
          <P className="mt-4 border-t border-border pt-3">
            Driver, host, database and user are fixed once registered. Only the password can be rotated.
          </P>
        </CardContent>
      </Card>

      {family === FAMILY.SQL && canRead && (
        <>
          {schemaPending ? (
            <Skeleton className="h-20 w-full" />
          ) : schemas.length === 0 ? (
            <p className="text-xs text-muted-foreground">No tables in this instance.</p>
          ) : (
            <div className="flex gap-2 overflow-x-auto pb-1">
              {schemas.map((s) => {
                const count = tables.filter((t) => t.schema === s).length;
                return (
                  <Link
                    key={s}
                    to={`${PAGE_ROUTES.INSTANCE_TAB(instance.id ?? "", "tables")}?schema=${encodeURIComponent(s)}`}
                    className="min-w-32 shrink-0 rounded-lg border p-3 text-left hover:bg-accent/50"
                  >
                    <div className="truncate font-mono text-xs">{s}</div>
                    <div className="mt-1 text-lg font-medium tabular-nums">{count}</div>
                    <div className="text-[10px] text-muted-foreground">table(s)</div>
                  </Link>
                );
              })}
            </div>
          )}
        </>
      )}
    </div>
  );
}
