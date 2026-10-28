import { useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ArrowDown, ArrowUp, Eraser, KeyRound, RefreshCw, Search, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ResultTable } from "@/components/sql/result-table";
import { useDebounce } from "@/hooks/use-debounce";
import { useAuth } from "@/app/auth-provider";
import { Workspace } from "@/components/layout/workspace";
import {
  useFlushMutation,
  useMemcacheConnections,
  useMemcacheKeysQuery,
  useMemcacheValueQuery,
  useMemcacheUrlParams,
  useRemoveKeysMutation,
} from "@/hooks/use-memcache";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

type SortKey = "name" | "type" | "ttl" | "size";

// The wire type marks every field optional; the browser works with a normalized row so sorting and selection stay simple.
interface Row {
  name: string;
  type: string;
  ttl: number;
  size: number;
}

// MemcachePage browses keys, inspects values, removes selections, and confirms database flushes by name.
export default function MemcachePage() {
  const { isInstanceAdmin } = useAuth();
  const { connections, connection, pick } = useMemcacheConnections();
  // removing keys and emptying the database are admin actions on the instance
  const mayAdmin = isInstanceAdmin(connection);
  const { pattern, inspected, setPattern, setInspected } = useMemcacheUrlParams();
  const glob = useDebounce(pattern || "*", 300);
  const selectionScope = JSON.stringify([connection, glob]);
  const [selection, setSelection] = useState<{ scope: string; keys: string[] }>({ scope: "", keys: [] });
  const selected = selection.scope === selectionScope ? selection.keys : [];
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({ key: "name", desc: false });
  const [flushOpen, setFlushOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset: resetFlush,
    formState: { errors: flushErrors },
  } = useForm<{ confirm: string }>({
    defaultValues: { confirm: "" },
    resolver: zodResolver(
      z.object({ confirm: z.string() }).superRefine((v, ctx) => {
        if (connection && v.confirm !== connection) {
          ctx.addIssue({ code: "custom", path: ["confirm"], message: "the name does not match — nothing is emptied" });
        }
      }),
    ),
  });

  const { page, isFetching: loading, refetch: reloadKeys } = useMemcacheKeysQuery(connection, glob);
  const { result, isFetching: valueLoading } = useMemcacheValueQuery(connection, inspected);

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

  const removeKeys = useRemoveKeysMutation();
  const onRemoveKeys = async () => {
    if (!connection || selected.length === 0) return;
    const res = await removeKeys.mutateAsync({ connection, keys: selected });
    toast.success(`${res.deleted} key(s) removed`);
    setSelection({ scope: selectionScope, keys: [] });
    if (inspected && selected.includes(inspected)) setInspected(undefined);
  };

  const flush = useFlushMutation();
  const onFlush = handleSubmit(async () => {
    if (!connection) return;
    await flush.mutateAsync(connection);
    toast.success(`${connection}: database emptied`);
    setFlushOpen(false);
    resetFlush();
    setSelection({ scope: selectionScope, keys: [] });
    setInspected(undefined);
  });

  function toggle(key: string) {
    const keys = selected.includes(key) ? selected.filter((item) => item !== key) : [...selected, key];
    setSelection({ scope: selectionScope, keys });
  }

  function head(key: SortKey, label: string) {
    const active = sort.key === key;
    return (
      <Button
        variant="ghost"
        size="sm"
        className={cn("flex items-center gap-1 px-1", active && "text-foreground")}
        onClick={() => setSort((prev) => ({ key, desc: prev.key === key ? !prev.desc : false }))}
      >
        {label}
        {active && (sort.desc ? <ArrowDown className="size-3" /> : <ArrowUp className="size-3" />)}
      </Button>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-col gap-1.5">
              <Label>Instance</Label>
              <Select
                value={connection ?? ""}
                onValueChange={(v) => {
                  setInspected(undefined);
                  pick(String(v));
                }}
              >
                <SelectTrigger className="w-full">
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
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="key-pattern">Key pattern</Label>
              <div className="relative">
                <Search className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id="key-pattern"
                  value={pattern}
                  onChange={(e) => {
                    setSelection({ scope: selectionScope, keys: [] });
                    setInspected(undefined);
                    setPattern(e.target.value);
                  }}
                  placeholder="session:*"
                  className="w-full pl-7"
                />
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Button variant="outline" size="sm" onClick={() => void reloadKeys()} disabled={loading || !connection}>
                <RefreshCw className={cn(loading && "animate-spin")} /> Refresh
              </Button>
              <Badge variant="outline">{page.total} key(s) in db</Badge>
            </div>

            {page.truncated && (
              <Tooltip>
                <TooltipTrigger render={<span className="text-xs text-amber-500" />}>scan truncated</TooltipTrigger>
                <TooltipContent>
                  The keyspace holds more keys than one scan reads (500 by default). Narrow the pattern.
                </TooltipContent>
              </Tooltip>
            )}
          </div>
        }
        right={
          keys.length === 0 ? (
            <Empty
              icon={KeyRound}
              title={connection ? "No keys" : "No memory cache selected"}
              description={connection ? "No key matches this pattern." : "Register a memory cache to browse its keys."}
            />
          ) : (
            <div className="min-h-0 flex-1 overflow-auto">
              <Table>
                <TableHeader className="sticky top-0 bg-card">
                  <TableRow>
                    <TableHead className="w-8" />
                    <TableHead>{head("name", "Key")}</TableHead>
                    <TableHead>{head("type", "Type")}</TableHead>
                    <TableHead>{head("ttl", "TTL")}</TableHead>
                    <TableHead>{head("size", "Size")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {keys.map((key) => (
                    <TableRow
                      key={key.name}
                      className={cn("cursor-pointer", inspected === key.name && "bg-accent/60")}
                      onClick={() => setInspected(key.name)}
                    >
                      <TableCell onClick={(e) => e.stopPropagation()}>
                        <Checkbox
                          aria-label={`select ${key.name}`}
                          checked={selected.includes(key.name)}
                          onCheckedChange={() => toggle(key.name)}
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
            </div>
          )
        }
        footer={
          <>
            <span>{selected.length} key(s) selected</span>
            {mayAdmin && (
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={selected.length === 0 || removeKeys.isPending}
                  onClick={() => void onRemoveKeys()}
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
          </>
        }
      />

      <Dialog open={inspected !== undefined} onOpenChange={(open) => !open && setInspected(undefined)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle className="truncate font-mono text-base">{inspected}</DialogTitle>
            <DialogDescription>rendered per the key's type</DialogDescription>
          </DialogHeader>
          <div className="max-h-[60svh] overflow-auto">
            {valueLoading ? (
              <P size="sm">loading…</P>
            ) : (
              <ResultTable result={result} title={inspected ?? "value"} />
            )}
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={flushOpen} onOpenChange={(open) => {
        setFlushOpen(open);
        if (!open) resetFlush();
      }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Empty the database</DialogTitle>
            <DialogDescription>
              Removes every key in the selected database index on {connection}. Other database indexes and other
              servers are untouched, and this cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <form id="flush-confirm" onSubmit={onFlush}>
            <Field>
              <FieldLabel htmlFor="flush-confirm">Confirm by typing the connection name</FieldLabel>
              <Input
                id="flush-confirm"
                placeholder={connection ?? "the connection name"}
                aria-invalid={!!flushErrors.confirm}
                {...register("confirm")}
              />
              <FieldError errors={[flushErrors.confirm]} />
            </Field>
          </form>
          <DialogFooter>
            <Button variant="destructive" type="submit" form="flush-confirm" disabled={flush.isPending}>
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
