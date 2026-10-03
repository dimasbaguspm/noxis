import { Area, AreaChart, CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { Radio } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ConnectionSelect } from "@/components/connection/connection-select";
import { useConnectionSelection } from "@/hooks/use-connection";
import { useStatsStream } from "@/hooks/use-data";

const AXIS = { stroke: "var(--muted-foreground)", fontSize: 11 };

// AnalyticsPage is the live instance view: the API streams a snapshot every
// 2 seconds (SSE) and recharts renders the series.
export default function AnalyticsPage() {
  const { connection, pick } = useConnectionSelection();
  const { latest, history, connected } = useStatsStream(connection);

  const samples = history.map((s, i) => ({
    i,
    at: new Date((s.takenAt ?? 0) * 1000).toLocaleTimeString(),
    connections: s.connections ?? 0,
    active: s.active ?? 0,
    idleInTransaction: s.idleInTransaction ?? 0,
    cacheHit: Number(((s.cacheHitRatio ?? 0) * 100).toFixed(2)),
    commits: s.commits ?? 0,
    rollbacks: s.rollbacks ?? 0,
    sizeMb: Number(((s.databaseSizeBytes ?? 0) / 1024 / 1024).toFixed(2)),
  }));

  const peak = samples.reduce((max, s) => Math.max(max, s.connections), 0);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="text-lg font-semibold">Analytics</h2>
        <ConnectionSelect value={connection} onChange={pick} />
        <Badge variant={connected ? "default" : "outline"}>
          <Radio className={connected ? "animate-pulse" : undefined} />
          {connected ? "live · 2s" : "connecting…"}
        </Badge>
        {latest?.version && (
          <span className="truncate text-xs text-muted-foreground" title={latest.version}>
            {latest.version.split(" on ")[0]}
          </span>
        )}
      </div>

      {!latest ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Waiting for the first snapshot…</CardTitle>
            <CardDescription>the stream starts as soon as the instance answers</CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Stat title="Connections" value={latest.connections ?? 0} hint={`session peak ${peak}`} />
            <Stat title="Active" value={latest.active ?? 0} hint={`${latest.idle ?? 0} idle`} />
            <Stat title="Max connections" value={latest.maxConnections ?? 0} hint="server setting" />
            <Stat title="Database size" value={`${samples.at(-1)?.sizeMb ?? 0} MB`} hint="current database" />
            <Stat title="Cache hit ratio" value={`${((latest.cacheHitRatio ?? 0) * 100).toFixed(1)}%`} hint="blocks from cache" />
            <Stat title="Longest query" value={`${(latest.longestQuerySeconds ?? 0).toFixed(2)} s`} hint="running now" />
            <Stat title="Commits" value={compact(latest.commits ?? 0)} hint="since server start" />
            <Stat
              title="Rollbacks / deadlocks"
              value={`${compact(latest.rollbacks ?? 0)} / ${latest.deadlocks ?? 0}`}
              hint={latest.uptimeSeconds ? `uptime ${duration(latest.uptimeSeconds)}` : ""}
            />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <ChartCard title="Connections" description="total, active and idle-in-transaction">
              <ResponsiveContainer width="100%" height={260}>
                <AreaChart data={samples}>
                  <defs>
                    <linearGradient id="conn" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#58a6ff" stopOpacity={0.6} />
                      <stop offset="95%" stopColor="#58a6ff" stopOpacity={0.05} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="at" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} allowDecimals={false} width={32} />
                  <Tooltip
                    contentStyle={{ background: "var(--popover)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }}
                  />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Area type="monotone" dataKey="connections" stroke="#58a6ff" fill="url(#conn)" />
                  <Line type="monotone" dataKey="active" stroke="#d29922" dot={false} />
                  <Line type="monotone" dataKey="idleInTransaction" stroke="#f85149" dot={false} />
                </AreaChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Cache hit ratio" description="percentage of blocks served from cache">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={samples}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="at" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} domain={[0, 100]} width={32} />
                  <Tooltip
                    contentStyle={{ background: "var(--popover)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }}
                  />
                  <Line type="monotone" dataKey="cacheHit" stroke="#3fb950" dot={false} name="cache hit %" />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>
          </div>
        </>
      )}
    </div>
  );
}

function ChartCard({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

function Stat({ title, value, hint }: { title: string; value: string | number; hint?: string }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{title}</CardDescription>
        <CardTitle className="text-2xl tabular-nums">{value}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-xs text-muted-foreground">{hint ?? "\u00a0"}</p>
      </CardContent>
    </Card>
  );
}

function compact(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

function duration(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days) return `${days}d ${hours}h`;
  if (hours) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}
