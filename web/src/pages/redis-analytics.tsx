import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Radio } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useRedisConnections, useRedisStatsStream } from "@/hooks/use-redis";

const AXIS = { stroke: "var(--muted-foreground)", fontSize: 11 };
const TOOLTIP = {
  contentStyle: {
    background: "var(--popover)",
    border: "1px solid var(--border)",
    borderRadius: 8,
    fontSize: 12,
  },
};

// RedisAnalyticsPage charts what INFO reports every 2 seconds.
export default function RedisAnalyticsPage() {
  const { connections, connection, pick } = useRedisConnections();
  const { latest, history, connected } = useRedisStatsStream(connection);

  const samples = history.map((s, i) => ({
    i,
    memoryMb: Number(((s.usedMemoryBytes ?? 0) / 1024 / 1024).toFixed(2)),
    peakMb: Number(((s.usedMemoryPeakBytes ?? 0) / 1024 / 1024).toFixed(2)),
    clients: s.connectedClients ?? 0,
    ops: s.opsPerSec ?? 0,
    hit: Number(((s.hitRatio ?? 0) * 100).toFixed(2)),
    keys: s.keys ?? 0,
    expires: s.expires ?? 0,
    evicted: s.evictedKeys ?? 0,
    expired: s.expiredKeys ?? 0,
  }));

  const hitRate = latest ? ((latest.hitRatio ?? 0) * 100).toFixed(1) : "0";
  const memoryLimit = latest?.maxMemoryBytes ? bytes(latest.maxMemoryBytes) : "no limit";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={connection ?? ""} onValueChange={(v) => pick(String(v))}>
          <SelectTrigger className="w-56">
            <SelectValue placeholder="no redis connection" />
          </SelectTrigger>
          <SelectContent>
            {connections.length === 0 && <SelectItem value="none">no connection yet</SelectItem>}
            {connections.map((c) => (
              <SelectItem key={c.name} value={c.name}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Badge variant={connected ? "default" : "outline"}>
          <Radio className={connected ? "animate-pulse" : undefined} />
          {connected ? "streaming" : "reconnecting…"}
        </Badge>
        {latest && (
          <span className="text-xs text-muted-foreground">
            {latest.version} · {latest.role} · {history.length} buffered
          </span>
        )}
      </div>

      {!latest ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Waiting for the first INFO snapshot…</CardTitle>
            <CardDescription>the stream starts as soon as the server answers</CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Stat title="Clients" value={latest.connectedClients ?? 0} hint={`${latest.totalConnectionsReceived ?? 0} accepted since start`} />
            <Stat title="Ops / sec" value={latest.opsPerSec ?? 0} hint={`${compact(latest.totalCommandsProcessed ?? 0)} commands total`} />
            <Stat title="Keys" value={compact(latest.keys ?? 0)} hint={`${compact(latest.expires ?? 0)} with a TTL`} />
            <Stat title="Memory" value={bytes(latest.usedMemoryBytes ?? 0)} hint={`peak ${bytes(latest.usedMemoryPeakBytes ?? 0)} · ${memoryLimit}`} />
            <Stat title="Hit ratio" value={`${hitRate}%`} hint={`${compact(latest.keyspaceHits ?? 0)} hits / ${compact(latest.keyspaceMisses ?? 0)} misses`} />
            <Stat title="Evicted / expired" value={`${compact(latest.evictedKeys ?? 0)} / ${compact(latest.expiredKeys ?? 0)}`} hint="since server start" />
            <Stat title="Fragmentation" value={(latest.memoryFragmentationRatio ?? 0).toFixed(2)} hint="used vs resident memory" />
            <Stat title="Uptime" value={duration(latest.uptimeSeconds ?? 0)} hint={`rdb ${latest.rdbLastSaveStatus ?? "?"} · aof ${latest.aofEnabled ? (latest.aofLastWriteStatus ?? "on") : "off"}`} />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <ChartCard title="Memory" description="used and peak, in MB">
              <ResponsiveContainer width="100%" height={260}>
                <AreaChart data={samples}>
                  <defs>
                    <linearGradient id="mem" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#58a6ff" stopOpacity={0.6} />
                      <stop offset="95%" stopColor="#58a6ff" stopOpacity={0.05} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} width={44} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Area type="monotone" dataKey="memoryMb" name="used MB" stroke="#58a6ff" fill="url(#mem)" />
                  <Line type="monotone" dataKey="peakMb" name="peak MB" stroke="#d29922" dot={false} />
                </AreaChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Throughput" description="ops per second and connected clients">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={samples}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} width={44} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Line type="monotone" dataKey="ops" name="ops/sec" stroke="#3fb950" dot={false} />
                  <Line type="monotone" dataKey="clients" name="clients" stroke="#a371f7" dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Cache" description="hit ratio and keyspace size">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={samples}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} yAxisId="pct" domain={[0, 100]} width={36} />
                  <YAxis {...AXIS} yAxisId="n" orientation="right" width={44} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Line yAxisId="pct" type="monotone" dataKey="hit" name="hit %" stroke="#3fb950" dot={false} />
                  <Line yAxisId="n" type="monotone" dataKey="keys" name="keys" stroke="#58a6ff" dot={false} />
                  <Line yAxisId="n" type="monotone" dataKey="expires" name="with TTL" stroke="#d29922" dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Pressure" description="evicted and expired keys">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={samples}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} width={44} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Line type="monotone" dataKey="evicted" name="evicted" stroke="#f85149" dot={false} />
                  <Line type="monotone" dataKey="expired" name="expired" stroke="#d29922" dot={false} />
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

function bytes(n: number): string {
  if (!n) return "0 B";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function duration(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days) return `${days}d ${hours}h`;
  if (hours) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}
