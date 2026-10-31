import { useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { ArrowDown, ArrowUp, Copy, Eraser, KeyRound, RefreshCw, Timer, Trash2 } from "lucide-react";
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
import { FilterField, SearchField } from "@/components/filters/filter-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { CacheStats } from "@/components/instance/cache-stats";
import { ResultTable } from "@/components/sql/result-table";
import { useDebounce } from "@/hooks/use-debounce";
import { useMemcacheBrowser, type SortKey } from "@/hooks/use-memcache-browser";
import { useAuth } from "@/app/auth-provider";
import { Workspace } from "@/components/layout/workspace";
import { QueuePanel } from "@/components/layout/queue-panel";
import {
  useFlushMutation,
  useMemcacheConnections,
  useMemcacheKeysQuery,
  useMemcacheExpireMutation,
  useMemcacheInfoQuery,
  useMemcacheValueQuery,
  useMemcacheUrlParams,
  useRemoveKeysMutation,
} from "@/hooks/use-memcache";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

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
  const { connections, connection, driver, pick } = useMemcacheConnections();
  // removing keys and emptying the database are admin actions on the instance
  const mayAdmin = isInstanceAdmin(connection ?? "");
  const { pattern, inspected, setPattern, setInspected } = useMemcacheUrlParams();
  const glob = useDebounce(pattern || "*", 300);
  const selectionScope = JSON.stringify([connection, glob]);
  const {
    selected,
    sort,
    sortBy,
    clearSelection,
    toggle,
    flushOpen,
    setFlushOpen,
    ttlOpen,
    setTtlOpen,
    ttlSeconds,
    setTtlSeconds,
  } = useMemcacheBrowser(selectionScope);
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
          ctx.addIssue({
            code: "custom",
            path: ["confirm"],
            message: "the name does not match — nothing is emptied",
          });
        }
      }),
    ),
  });

  const {
    page,
    isFetching: loading,
    refetch: reloadKeys,
    loadMore,
  } = useMemcacheKeysQuery(driver, connection, glob);
  const { result, isFetching: valueLoading } = useMemcacheValueQuery(driver, connection, inspected);
  const { info } = useMemcacheInfoQuery(driver, connection);

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
    if (!driver || !connection || selected.length === 0) return;
    const res = await removeKeys.mutateAsync({ driver, connection, keys: selected });
    toast.success(`${res.deleted} key(s) removed`);
    clearSelection();
    if (inspected && selected.includes(inspected)) setInspected(undefined);
  };

  const flush = useFlushMutation();
  const expire = useMemcacheExpireMutation(driver, connection);
  const onFlush = handleSubmit(async () => {
    if (!driver || !connection) return;
    await flush.mutateAsync({ driver, connection });
    toast.success(`${connection}: database emptied`);
    setFlushOpen(false);
    resetFlush();
    clearSelection();
    setInspected(undefined);
  });

  function head(key: SortKey, label: string) {
    const active = sort.key === key;
    return (
      <Button
        variant="ghost"
        size="sm"
        className={cn("flex items-center gap-1 px-1", active && "text-foreground")}
        onClick={() => sortBy(key)}
      >
        {label}
        {active && (sort.desc ? <ArrowDown className="size-3" /> : <ArrowUp className="size-3" />)}
      </Button>
    );
  }

  // the inspected row carries the metadata the value endpoint does not repeat
  const inspectedRow = keys.find((k) => k.name === inspected);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <Workspace
        footer={
          <>
            <span>{selected.length} key(s) selected</span>
            <span>
              {page.total} scanned{page.truncated ? " · scan truncated" : ""}
            </span>
            <CacheStats info={info} />
            {page.truncated && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => loadMore.mutate()}
                disabled={loadMore.isPending}
              >
                Load more
              </Button>
            )}
          </>
        }
      >
        <Workspace.Pane defaultSize={17} minSize={14}>
          <QueuePanel
            actions={
              <>
                <Button
                  variant="outline"
                  onClick={() => void reloadKeys()}
                  disabled={loading || !connection}
                >
                  <RefreshCw className={cn(loading && "animate-spin")} /> Refresh
                </Button>
                {mayAdmin && (
                  <Button
                    variant="outline"
                    disabled={selected.length === 0 || removeKeys.isPending}
                    onClick={() => void onRemoveKeys()}
                  >
                    <Trash2 /> Remove selected ({selected.length})
                  </Button>
                )}
                {mayAdmin && (
                  <Button
                    variant="outline"
                    disabled={selected.length === 0 || expire.isPending}
                    onClick={() => setTtlOpen(true)}
                  >
                    <Timer /> Expire ({selected.length})
                  </Button>
                )}
                {mayAdmin && (
                  <Button
                    variant="ghost"
                    className="text-destructive"
                    disabled={!connection}
                    onClick={() => setFlushOpen(true)}
                  >
                    <Eraser /> Empty database
                  </Button>
                )}
              </>
            }
          >
            <FilterField label="Instance">
              <Select
                value={connection ?? ""}
                onValueChange={(v) => {
                  setInspected(undefined);
                  pick(String(v));
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="No memory cache registered" />
                </SelectTrigger>
                <SelectContent>
                  {connections.length === 0 && (
                    <SelectItem value="none">None registered yet</SelectItem>
                  )}
                  {connections.map((c) => (
                    <SelectItem key={c.name} value={c.name}>
                      {c.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FilterField>

            <SearchField
              id="key-pattern"
              label="Key pattern"
              placeholder="session:*"
              value={pattern}
              onChange={(value) => {
                clearSelection();
                setInspected(undefined);
                setPattern(value);
              }}
            />

            {page.truncated && (
              <p className="text-xs text-amber-500">
                The keyspace holds more keys than one scan reads (500 by default). Narrow the
                pattern.
              </p>
            )}
          </QueuePanel>
        </Workspace.Pane>

        <Workspace.Split defaultSize={83} minSize={40} direction="horizontal">
          <Workspace.Pane defaultSize={68} minSize={35} padded={false} scroll>
            {keys.length === 0 ? (
              <Empty
                icon={KeyRound}
                title={connection ? "No keys" : "No memory cache selected"}
                description={
                  connection
                    ? "No key matches this pattern."
                    : "Register a memory cache to browse its keys."
                }
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
                        <TableCell className="text-xs text-muted-foreground">
                          {ttl(key.ttl)}
                        </TableCell>
                        <TableCell className="text-xs tabular-nums text-muted-foreground">
                          {key.size ? bytes(key.size) : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </Workspace.Pane>

          <Workspace.Pane defaultSize={32} minSize={20} scroll>
            {inspected === undefined ? (
              <P size="sm" className="text-muted-foreground">
                Pick a key to read it here, beside the list.
              </P>
            ) : (
              <div className="flex min-h-0 flex-1 flex-col gap-2">
                <div className="flex items-center gap-2">
                  <span className="truncate font-mono text-xs" title={inspected}>
                    {inspected}
                  </span>
                  <span className="flex-1" />
                  <Button
                    variant="ghost"
                    size="xs"
                    onClick={() => void navigator.clipboard.writeText(inspected)}
                  >
                    <Copy /> Copy
                  </Button>
                </div>
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <Badge variant="outline" className="text-[10px]">
                    {inspectedRow?.type ?? "—"}
                  </Badge>
                  <span>{inspectedRow ? ttl(inspectedRow.ttl) : "—"}</span>
                  <span>{inspectedRow?.size ? bytes(inspectedRow.size) : "—"}</span>
                </div>
                <div className="min-h-0 flex-1 overflow-auto">
                  {valueLoading ? (
                    <P size="sm">Loading…</P>
                  ) : (
                    <ResultTable result={result} title={inspected} />
                  )}
                </div>
              </div>
            )}
          </Workspace.Pane>
        </Workspace.Split>
      </Workspace>

      <Dialog open={ttlOpen} onOpenChange={setTtlOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Set or clear an expiry</DialogTitle>
            <DialogDescription>
              A TTL is metadata, so this is applied directly: enter seconds to set one, or 0 to
              clear it. The keys are not touched otherwise.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <p className="text-xs text-muted-foreground">{selected.length} key(s) selected</p>
            <Input
              value={ttlSeconds}
              onChange={(event) => setTtlSeconds(event.target.value)}
              inputMode="numeric"
              placeholder="seconds, 0 clears the expiry"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setTtlOpen(false)}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => {
                expire.mutate({ keys: selected, ttlMs: Number(ttlSeconds) * 1000 });
                setTtlOpen(false);
              }}
              disabled={!Number.isFinite(Number(ttlSeconds)) || expire.isPending}
            >
              <Timer /> Apply
            </Button>
          </DialogFooter>
          {expire.isError && <p className="text-xs text-destructive">{expire.error.message}</p>}
        </DialogContent>
      </Dialog>

      <Dialog
        open={flushOpen}
        onOpenChange={(open) => {
          setFlushOpen(open);
          if (!open) resetFlush();
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Empty the database</DialogTitle>
            <DialogDescription>
              Removes every key in the selected database index on {connection}. Other database
              indexes and other servers are untouched, and this cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <form id="flush-confirm" onSubmit={onFlush}>
            <Field>
              <FieldLabel htmlFor="flush-confirm">Confirm by typing the connection name</FieldLabel>
              <Input
                id="flush-confirm"
                placeholder={connection ?? "The connection name"}
                aria-invalid={!!flushErrors.confirm}
                {...register("confirm")}
              />
              <FieldError errors={[flushErrors.confirm]} />
            </Field>
          </form>
          <DialogFooter>
            <Button
              variant="destructive"
              type="submit"
              form="flush-confirm"
              disabled={flush.isPending}
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
