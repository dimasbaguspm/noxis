import { Activity, Radio } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useApp } from "@/app/app-provider";
import { useStatsStream } from "@/hooks/use-data";
import type { Stats } from "@/lib/api";

// AnalyticsPage is the live instance view: the API streams a snapshot every
// 2 seconds (SSE) and this page renders it as tiles plus sparklines.
export default function AnalyticsPage() {
  const { active } = useApp();
  const { latest, history, connected } = useStatsStream(active);
  const peakConnections = history.reduce((max, s) => Math.max(max, s.connections ?? 0), 0);

  if (!active) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>No connection selected</CardTitle>
          <CardDescription>pick a connection to watch it live</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <h2 className="text-lg font-semibold">{active}</h2>
        <Badge variant={connected ? "default" : "outline"}>
          <Radio className={connected ? "animate-pulse" : undefined} />
          {connected ? "live · 2s" : "connecting…"}
        </Badge>
        {latest?.version && <span className="truncate text-xs text-muted-foreground">{latest.version}</span>}
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
            <Tile title="Connections" value={latest.connections ?? 0} series={series(history, (s) => s.connections)} />
            <Tile title="Peak (this session)" value={peakConnections} hint="derived from the stream" />
            <Tile title="Active" value={latest.active ?? 0} series={series(history, (s) => s.active)} tone="text-amber-500" />
            <Tile title="Max connections" value={latest.maxConnections ?? 0} hint="server setting" />
            <Tile
              title="Database size"
              value={bytes(latest.databaseSizeBytes ?? 0)}
              series={series(history, (s) => s.databaseSizeBytes)}
            />
            <Tile title="Cache hit ratio" value={pct(latest.cacheHitRatio)} hint="blocks served from cache" />
            <Tile title="Longest query" value={`${(latest.longestQuerySeconds ?? 0).toFixed(1)} s`} hint="currently running" />
            <Tile title="Commits" value={compact(latest.commits ?? 0)} hint="since server start" />
            <Tile title="Rollbacks / deadlocks" value={`${compact(latest.rollbacks ?? 0)} / ${latest.deadlocks ?? 0}`} />
            {latest.uptimeSeconds ? <Tile title="Uptime" value={duration(latest.uptimeSeconds)} /> : null}
            {latest.tempBytes ? <Tile title="Temp bytes" value={bytes(latest.tempBytes)} hint="spilled to disk" /> : null}
            {latest.idleInTransaction !== undefined && (
              <Tile title="Idle in transaction" value={latest.idleInTransaction} hint="watch for lock holders" />
            )}
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <Activity className="size-4" /> Connections over the last {history.length} samples
              </CardTitle>
              <CardDescription>streamed from /v1/connections/{active}/stats/stream</CardDescription>
            </CardHeader>
            <CardContent>
              <Sparkline values={series(history, (s) => s.connections)} height={64} />
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function series(history: Stats[], pick: (s: Stats) => number | undefined): number[] {
  return history.map((s) => pick(s) ?? 0);
}

function Tile({
  title,
  value,
  hint,
  tone,
  series: values,
}: {
  title: string;
  value: string | number;
  hint?: string;
  tone?: string;
  series?: number[];
}) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{title}</CardDescription>
        <CardTitle className={`text-2xl tabular-nums ${tone ?? ""}`}>{value}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-1">
        {values && values.length > 1 && <Sparkline values={values} height={28} />}
        <p className="text-xs text-muted-foreground">{hint ?? "\u00a0"}</p>
      </CardContent>
    </Card>
  );
}

// Sparkline is a pure SVG polyline — no chart dependency for a 60-point trend.
function Sparkline({ values, height }: { values: number[]; height: number }) {
  if (values.length < 2) return <div style={{ height }} />;
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const span = max - min || 1;
  const points = values
    .map((v, i) => {
      const x = (i / (values.length - 1)) * 100;
      const y = height - ((v - min) / span) * height;
      return `${x.toFixed(2)},${y.toFixed(2)}`;
    })
    .join(" ");
  return (
    <svg viewBox={`0 0 100 ${height}`} preserveAspectRatio="none" className="w-full" style={{ height }}>
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function bytes(n: number): string {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value >= 10 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`;
}

function compact(n: number): string {
  return Intl.NumberFormat("en", { notation: "compact" }).format(n);
}

function pct(ratio?: number): string {
  return ratio === undefined ? "—" : `${(ratio * 100).toFixed(1)}%`;
}

function duration(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days) return `${days}d ${hours}h`;
  if (hours) return `${hours}h ${minutes}m`;
  return `${minutes}m`;
}
