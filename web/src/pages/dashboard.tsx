import { Link } from "react-router-dom";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ConnectionList } from "@/components/connection/connection-list";
import { useApp } from "@/app/app-provider";
import { useChanges } from "@/hooks/use-data";

export default function DashboardPage() {
  const { connections, schema, active } = useApp();
  const { total, changes } = useChanges();

  const byStatus = (statuses: string[]) => changes.filter((c) => statuses.includes(c.status ?? "")).length;
  const applied = byStatus(["applied"]);
  const pending = byStatus(["validated", "approved"]);
  const failed = byStatus(["failed"]);

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <Stat title="Connections" value={connections.length} hint={active ? `active: ${active}` : "none active"} />
        <Stat title="Tables" value={schema.length} hint={active ? `in ${active}` : "pick a connection"} />
        <Stat title="Pending changes" value={pending} hint="validated or approved" />
        <Stat title="Applied / failed" value={`${applied} / ${failed}`} hint={`${total} changes total`} />
      </div>

      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">Connections</h2>
        <span className="flex-1" />
        <Link to="/query" className="text-sm text-muted-foreground hover:underline">
          go to query →
        </Link>
      </div>

      <ConnectionList />
    </div>
  );
}

function Stat({ title, value, hint }: { title: string; value: string | number; hint: string }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{title}</CardDescription>
        <CardTitle className="text-2xl tabular-nums">{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}
