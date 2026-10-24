import { useState } from "react";
import { Link } from "react-router-dom";
import { ExternalLink, KeyRound, RefreshCw, Search } from "lucide-react";
import { ResultTable } from "@/components/sql/result-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Workspace } from "@/components/layout/workspace";
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
    <div className="flex h-full min-h-0">
      <Workspace
        leftLabel="Cache filters"
        rightLabel="Cache value"
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-2">
            <div className="relative">
              <Search className="pointer-events-none absolute top-2 left-2 size-3.5 text-muted-foreground" />
              <Input
                value={pattern}
                onChange={(e) => setPattern(e.target.value)}
                placeholder="key pattern"
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
            <Badge variant="outline" className="w-fit">{keys.page.total ?? 0} key(s)</Badge>
            <div className="min-h-0 flex-1 overflow-auto">
              {visible.length === 0 ? (
                <P size="sm">{keys.isFetching ? "Loading keys…" : "No key matches this pattern."}</P>
              ) : (
                <div className="grid gap-0.5">
                  {visible.map((item) => (
                    <button
                      key={item.name}
                      type="button"
                      onClick={() => setKey(item.name)}
                      className={cn(
                        "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs hover:bg-accent/60",
                        key === item.name && "bg-accent text-accent-foreground",
                      )}
                    >
                      <span className="min-w-0 flex-1 truncate font-mono">{item.name}</span>
                      <Badge variant="outline" className="shrink-0 text-[10px]">{item.type}</Badge>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        }
        right={
          !key ? (
            <Empty icon={KeyRound} title="No key selected" description="Pick a key on the left to inspect its value." />
          ) : value.isFetching ? (
            <P size="sm">Loading value…</P>
          ) : value.error ? (
            <p className="text-xs text-destructive">
              {value.error instanceof Error ? value.error.message : "the value could not load"}
            </p>
          ) : (
            <div className="min-h-0 flex-1 overflow-auto">
              <ResultTable result={value.result} title={key} />
            </div>
          )
        }
      />
    </div>
  );
}
