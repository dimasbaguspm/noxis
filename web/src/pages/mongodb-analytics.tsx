import { useEffect, useState } from "react";
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
import { useMongoCollections, useMongoConnections, useMongoDatabases, useMongoStatsStream } from "@/hooks/use-mongodb";
import { bytes, compact, duration } from "@/lib/format";

const AXIS = { stroke: "var(--muted-foreground)", fontSize: 11 };
const TOOLTIP = {
  contentStyle: {
    background: "var(--popover)",
    border: "1px solid var(--border)",
    borderRadius: 8,
    fontSize: 12,
  },
};

// MongoAnalyticsPage watches a server, a database and a collection: the
// operation counters, memory and network from serverStatus, the sizes from
// dbStats, and the collection's own count and index sizes.
export default function MongoAnalyticsPage() {
  const { connections, connection, pick } = useMongoConnections();
  const { databases } = useMongoDatabases(connection);
  const [database, setDatabase] = useState<string>();
  const [collection, setCollection] = useState<string>();

  useEffect(() => {
    if (!database && databases.length > 0) setDatabase(databases[0]?.name);
  }, [database, databases]);

  const { collections } = useMongoCollections(connection, database);
  const { latest, history, connected } = useMongoStatsStream(connection, database, collection);

  const samples = history.map((s, i) => ({
    i,
    residentMb: s.residentMemoryMb ?? 0,
    virtualMb: s.virtualMemoryMb ?? 0,
    connections: s.connections ?? 0,
    ops: s.currentOps ?? 0,
    inserted: s.inserted ?? 0,
    queried: s.queried ?? 0,
    updated: s.updated ?? 0,
    deleted: s.deleted ?? 0,
  }));

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={connection ?? ""} onValueChange={(v) => pick(String(v))}>
          <SelectTrigger className="w-52">
            <SelectValue placeholder="no mongodb registered" />
          </SelectTrigger>
          <SelectContent>
            {connections.length === 0 && <SelectItem value="none">none registered yet</SelectItem>}
            {connections.map((c) => (
              <SelectItem key={c.name} value={c.name}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={database ?? ""} onValueChange={(v) => setDatabase(String(v))}>
          <SelectTrigger className="w-44">
            <SelectValue placeholder="database" />
          </SelectTrigger>
          <SelectContent>
            {databases.map((db) => (
              <SelectItem key={db.name} value={db.name ?? ""}>
                {db.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={collection ?? "all"} onValueChange={(v) => setCollection(String(v) === "all" ? undefined : String(v))}>
          <SelectTrigger className="w-52">
            <SelectValue placeholder="whole database" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">whole database</SelectItem>
            {collections.map((c) => (
              <SelectItem key={c.name} value={c.name ?? ""}>
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
            mongodb {latest.version} · {history.length} buffered
          </span>
        )}
      </div>

      {!latest ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Waiting for the first snapshot…</CardTitle>
            <CardDescription>the stream starts as soon as the server answers</CardDescription>
          </CardHeader>
        </Card>
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            <Stat title="Connections" value={latest.connections ?? 0} hint={`uptime ${duration(latest.uptimeSeconds ?? 0)}`} />
            <Stat title="Current ops" value={latest.currentOps ?? 0} hint="in flight right now" />
            <Stat
              title="Operations"
              value={compact((latest.inserted ?? 0) + (latest.queried ?? 0) + (latest.updated ?? 0) + (latest.deleted ?? 0))}
              hint={`${compact(latest.queried ?? 0)} queries · ${compact(latest.inserted ?? 0)} inserts`}
            />
            <Stat title="Resident memory" value={bytes((latest.residentMemoryMb ?? 0) * 1024 * 1024)} hint={`virtual ${latest.virtualMemoryMb ?? 0} MB`} />
            <Stat
              title={`${latest.database || "database"} size`}
              value={bytes(latest.dbDataSizeBytes ?? 0)}
              hint={`${compact(latest.dbObjects ?? 0)} docs · ${latest.dbCollections ?? 0} collections`}
            />
            <Stat
              title="Indexes"
              value={compact(latest.dbIndexes ?? 0)}
              hint={`${bytes(latest.dbIndexSizeBytes ?? 0)} on disk`}
            />
            <Stat
              title="Collection documents"
              value={latest.collection ? compact(latest.collectionCount ?? 0) : "—"}
              hint={latest.collection ? `${bytes(latest.collectionSizeBytes ?? 0)} data` : "pick a collection"}
            />
            <Stat
              title="Collection indexes"
              value={latest.collection ? compact(latest.collectionIndexes ?? 0) : "—"}
              hint={latest.collection ? `${bytes(latest.collectionIndexSizeBytes ?? 0)} · avg doc ${bytes(latest.collectionAvgObjectSizeBytes ?? 0)}` : "pick a collection"}
            />
          </div>

          <div className="grid gap-4 xl:grid-cols-2">
            <ChartCard title="Memory" description="resident and virtual, in MB">
              <ResponsiveContainer width="100%" height={260}>
                <AreaChart data={samples}>
                  <defs>
                    <linearGradient id="mongo-mem" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#58a6ff" stopOpacity={0.6} />
                      <stop offset="95%" stopColor="#58a6ff" stopOpacity={0.05} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} width={44} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Area type="monotone" dataKey="residentMb" name="resident MB" stroke="#58a6ff" fill="url(#mongo-mem)" />
                  <Line type="monotone" dataKey="virtualMb" name="virtual MB" stroke="#d29922" dot={false} />
                </AreaChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Traffic" description="connections and in-flight operations">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={samples}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} width={44} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Line type="monotone" dataKey="connections" name="connections" stroke="#a371f7" dot={false} />
                  <Line type="monotone" dataKey="ops" name="current ops" stroke="#3fb950" dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Operation counters" description="cumulative inserts, queries, updates and deletes">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={samples}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                  <XAxis dataKey="i" {...AXIS} minTickGap={24} />
                  <YAxis {...AXIS} width={52} />
                  <Tooltip {...TOOLTIP} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Line type="monotone" dataKey="queried" name="queries" stroke="#58a6ff" dot={false} />
                  <Line type="monotone" dataKey="inserted" name="inserts" stroke="#3fb950" dot={false} />
                  <Line type="monotone" dataKey="updated" name="updates" stroke="#d29922" dot={false} />
                  <Line type="monotone" dataKey="deleted" name="deletes" stroke="#f85149" dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </ChartCard>

            <ChartCard title="Storage" description="data, storage and index sizes">
              <div className="grid gap-3 sm:grid-cols-3">
                <Metric label="data" value={bytes(latest.dbDataSizeBytes ?? 0)} />
                <Metric label="storage" value={bytes(latest.dbStorageBytes ?? 0)} />
                <Metric label="indexes" value={bytes(latest.dbIndexSizeBytes ?? 0)} />
              </div>
              {latest.collection && (
                <>
                  <p className="mt-4 text-xs text-muted-foreground">
                    {latest.database}.{latest.collection}
                    {latest.collectionCapped ? " (capped)" : ""}
                  </p>
                  <div className="mt-2 grid gap-3 sm:grid-cols-3">
                    <Metric label="documents" value={compact(latest.collectionCount ?? 0)} />
                    <Metric label="data" value={bytes(latest.collectionSizeBytes ?? 0)} />
                    <Metric label="indexes" value={bytes(latest.collectionIndexSizeBytes ?? 0)} />
                  </div>
                </>
              )}
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

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-medium tabular-nums">{value}</p>
    </div>
  );
}
