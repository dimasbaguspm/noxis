import { useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Eraser, RefreshCw, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ResultTable } from "@/components/sql/result-table";
import { useAuth } from "@/app/auth-provider";
import { useAction } from "@/hooks/use-action";
import { useMemcacheConnections, useMemcacheKeys, useMemcacheValue } from "@/hooks/use-memcache";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";

type SortKey = "name" | "type" | "ttl" | "size";

// The wire type marks every field optional; the browser works with a
// normalized row so sorting and selection stay simple.
interface Row {
  name: string;
  type: string;
  ttl: number;
  size: number;
}

// MemcachePage browses a live memory-cache server: keys with their type and
// TTL, the value of a key in a dialog, removal of the selected keys, and
// emptying the database behind a typed confirmation. New instances are
// registered from the dashboard, the single entry point.
export default function MemcachePage() {
  const { can } = useAuth();
  const { connections, connection, pick } = useMemcacheConnections();
  // removing keys and emptying the database are admin actions on the instance
  const mayAdmin = can(connection, "admin");
  const [pattern, setPattern] = useState("*");
  const [glob, setGlob] = useState("*");
  const [selected, setSelected] = useState<string[]>([]);
  const [inspected, setInspected] = useState<string>();
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({ key: "name", desc: false });
  const [flushOpen, setFlushOpen] = useState(false);
  const [confirm, setConfirm] = useState("");

  // debounce the pattern so typing does not scan on every keystroke
  useEffect(() => {
    const id = window.setTimeout(() => setGlob(pattern || "*"), 300);
    return () => window.clearTimeout(id);
  }, [pattern]);

  const { page, loading, reload: reloadKeys } = useMemcacheKeys(connection, glob);
  const { result, loading: valueLoading } = useMemcacheValue(connection, inspected);

  useEffect(() => {
    setSelected([]);
    setInspected(undefined);
  }, [connection, glob]);

  const keys = useMemo<Row[]>(() => {
    const rows: Row[] = (page.keys ?? [])
      .map((k) => ({
        name: k.name ?? "",
        type: k.type ?? "",
        ttl: k.ttlMs ?? -1,
        size: k.sizeBytes ?? 0,
      }))
      .filter((k) => k.name !== "");
    const dir = sort.desc ? -1 : 1;
    return rows.sort((a, b) => {
      const [x, y] = [a[sort.key], b[sort.key]];
      if (typeof x === "number" && typeof y === "number") return (x - y) * dir;
      return String(x).localeCompare(String(y)) * dir;
    });
  }, [page.keys, sort]);

  const removeKeys = useAction(async () => {
    if (!connection || selected.length === 0) return;
    const res = await api.memcache.removeKeys(connection, selected);
    toast.success(`${res.deleted} key(s) removed`);
    setSelected([]);
    if (inspected && selected.includes(inspected)) setInspected(undefined);
    reloadKeys();
  });

  const flush = useAction(async () => {
    if (!connection) return;
    await api.memcache.flush(connection);
    toast.success(`${connection}: database emptied`);
    setFlushOpen(false);
    setConfirm("");
    setSelected([]);
    setInspected(undefined);
    reloadKeys();
  });

  function toggle(key: string) {
    setSelected((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  }

  function head(key: SortKey, label: string) {
    const active = sort.key === key;
    return (
      <button
        className={cn("flex items-center gap-1", active && "text-foreground")}
        onClick={() => setSort((prev) => ({ key, desc: prev.key === key ? !prev.desc : false }))}
      >
        {label}
        {active && (sort.desc ? <ArrowDown className="size-3" /> : <ArrowUp className="size-3" />)}
      </button>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={connection ?? ""} onValueChange={(v) => pick(String(v))}>
          <SelectTrigger className="w-56">
            <SelectValue placeholder="no memory cache registered" />
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
        <div className="flex items-center gap-2">
          <Search className="size-4 text-muted-foreground" />
          <Input
            value={pattern}
            onChange={(e) => setPattern(e.target.value)}
            placeholder="key pattern, e.g. session:*"
            className="w-64"
          />
        </div>
        <Button variant="outline" size="sm" onClick={reloadKeys} disabled={loading || !connection}>
          <RefreshCw className={cn(loading && "animate-spin")} /> Refresh
        </Button>
        <Badge variant="outline">{page.total} key(s) in db</Badge>
        {page.truncated && (
          <Tooltip>
            <TooltipTrigger render={<span className="text-xs text-amber-500" />}>scan truncated</TooltipTrigger>
            <TooltipContent>
              The keyspace holds more keys than one scan reads (500 by default). Narrow the pattern.
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      <Card className="flex min-h-0 flex-col">
        <CardHeader>
          <CardTitle className="text-base">Keys</CardTitle>
          <CardDescription>
            {selected.length > 0 ? `${selected.length} selected` : "click a key to open its value"}
          </CardDescription>
        </CardHeader>
        <CardContent className="min-h-0 overflow-auto">
          {mayAdmin && (
            <div className="mb-2 flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={selected.length === 0 || removeKeys.loading}
                onClick={() => void removeKeys.run()}
              >
                <Trash2 /> Remove selected
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="text-destructive"
                disabled={!connection}
                onClick={() => setFlushOpen(true)}
              >
                <Eraser /> Empty database
              </Button>
            </div>
          )}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
                <TableHead>{head("name", "Key")}</TableHead>
                <TableHead>{head("type", "Type")}</TableHead>
                <TableHead>{head("ttl", "TTL")}</TableHead>
                <TableHead>{head("size", "Size")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {keys.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-muted-foreground">
                    {connection ? "no key matches" : "register a memory cache from the dashboard to browse"}
                  </TableCell>
                </TableRow>
              )}
              {keys.map((key) => (
                <TableRow
                  key={key.name}
                  className={cn("cursor-pointer", inspected === key.name && "bg-accent/60")}
                  onClick={() => setInspected(key.name)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <input
                      type="checkbox"
                      aria-label={`select ${key.name}`}
                      checked={selected.includes(key.name)}
                      onChange={() => toggle(key.name)}
                    />
                  </TableCell>
                  <TableCell className="max-w-96 truncate font-mono text-xs" title={key.name}>
                    {key.name}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="text-[10px]">
                      {key.type}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{ttl(key.ttl)}</TableCell>
                  <TableCell className="text-xs tabular-nums text-muted-foreground">
                    {key.size ? bytes(key.size) : "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={inspected !== undefined} onOpenChange={(open) => !open && setInspected(undefined)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle className="truncate font-mono text-base">{inspected}</DialogTitle>
            <DialogDescription>rendered per the key's type</DialogDescription>
          </DialogHeader>
          <div className="max-h-[60svh] overflow-auto">
            {valueLoading ? (
              <p className="text-sm text-muted-foreground">loading…</p>
            ) : (
              <ResultTable result={result} title={inspected ?? "value"} />
            )}
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={flushOpen} onOpenChange={setFlushOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Empty the database</DialogTitle>
            <DialogDescription>
              Removes every key in the selected database index on {connection}. Other database indexes and other
              servers are untouched, and this cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <Input
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            placeholder={`type ${connection ?? "the connection name"} to confirm`}
          />
          <DialogFooter>
            <Button
              variant="destructive"
              disabled={confirm !== connection || flush.loading}
              onClick={() => void flush.run()}
            >
              <Eraser /> Empty database
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ttl renders milliseconds; -1 is Redis for "no expiry".
function ttl(ms: number): string {
  if (ms < 0) return "no expiry";
  if (ms < 1000) return `${ms} ms`;
  const seconds = Math.round(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.round(hours / 24)}d`;
}

function bytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}
