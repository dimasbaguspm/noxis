import { useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, RefreshCw, Search } from "lucide-react";
import { ResultTable } from "@/components/sql/result-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useDebounce } from "@/hooks/use-debounce";
import { useMemcacheConnections, useMemcacheKeysQuery, useMemcacheValueQuery } from "@/hooks/use-memcache";
import { PAGE_ROUTES } from "@/lib/routes";
import { P } from "@/components/ui/typography";
import { cn } from "@/lib/utils";

// CacheBrowser is a compact read-only key browser on the instance page; removal and flush stay on the full page.
export function CacheBrowser({ connection }: { connection: string }) {
  const [pattern, setPattern] = useState("*");
  const [key, setKey] = useState<string>();
  const glob = useDebounce(pattern.trim() || "*", 300);
  const keys = useMemcacheKeysQuery(connection, glob);
  const value = useMemcacheValueQuery(connection, key);
  const { pick } = useMemcacheConnections();
  const visible = (keys.page.keys ?? []).slice(0, 100);

  return (
    <Card>
      <CardHeader className="gap-1">
        <CardTitle className="text-base">Keys</CardTitle>
        <CardDescription>
          Browse the cache by pattern and inspect one value. Removal and flush stay on the full page.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative min-w-48 flex-1">
            <Search className="pointer-events-none absolute top-2 left-2 size-3.5 text-muted-foreground" />
            <Input
              value={pattern}
              onChange={(e) => setPattern(e.target.value)}
              placeholder="key pattern, e.g. session:*"
              className="h-8 pl-7 font-mono text-xs"
              aria-label="key pattern"
            />
          </div>
          <Button size="sm" variant="outline" onClick={() => void keys.refetch()} disabled={keys.isFetching}>
            <RefreshCw className={cn(keys.isFetching && "animate-spin")} /> Refresh
          </Button>
          <Button
            size="sm"
            variant="outline"
            render={<Link to={PAGE_ROUTES.MEMCACHE.KEYS} onClick={() => pick(connection)} />}
          >
            <ExternalLink /> Full page
          </Button>
          <Badge variant="outline">{keys.page.total ?? 0} key(s)</Badge>
        </div>

        <div className="grid gap-3 lg:grid-cols-2">
          <div className="h-72 overflow-auto rounded-md border border-border">
            <Table>
              <TableHeader className="sticky top-0 bg-card">
                <TableRow>
                  <TableHead>Key</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>TTL</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {visible.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} className="text-muted-foreground">
                      {keys.isFetching ? "Loading keys…" : "No key matches this pattern."}
                    </TableCell>
                  </TableRow>
                )}
                {visible.map((k) => (
                  <TableRow
                    key={k.name}
                    className={cn("cursor-pointer", key === k.name && "bg-accent/60")}
                    onClick={() => setKey(k.name)}
                  >
                    <TableCell className="max-w-64 truncate font-mono text-xs" title={k.name}>
                      {k.name}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="text-[10px]">
                        {k.type}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {k.ttlMs && k.ttlMs > 0 ? `${Math.ceil(k.ttlMs / 1000)}s` : "—"}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          <div className="h-72 overflow-auto rounded-md border border-border p-2">
            {!key ? (
              <P size="sm">Pick a key to inspect its value.</P>
            ) : value.isFetching ? (
              <P size="sm">Loading value…</P>
            ) : value.error ? (
              <p className="text-xs text-destructive">
                {value.error instanceof Error ? value.error.message : "the value could not load"}
              </p>
            ) : (
              <ResultTable result={value.result} title={key} />
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
