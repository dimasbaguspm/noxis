import { Link } from "react-router-dom";
import { Play, Search as SearchIcon, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Empty } from "@/components/ui/empty";
import { Textarea } from "@/components/ui/textarea";
import { JsonTree } from "@/components/mongodb/json-tree";
import { useDriverConnections } from "@/hooks/use-driver-connections";
import { useSearchRaise } from "@/hooks/use-search-changes";
import { useSearchMutation, useSearchTargetsQuery, useSearchUrlParams } from "@/hooks/use-search";
import { PAGE_ROUTES } from "@/constants/routes";
import { bytes, compact } from "@/lib/format";
import { cn } from "@/lib/utils";

// parseHit reads one hit the server already validated as JSON; a document that does not parse is shown as text rather
// than crashing the pane.
function parseHit(doc: string): unknown {
  try {
    return JSON.parse(doc);
  } catch {
    return doc;
  }
}

// SearchPage is the console for a search platform: what it indexes on the left, the platform's own query document and
// its hits on the right. The payload is the platform's language, so the editor is text, not a form.
export default function SearchPage() {
  const { connections, connection, driver, pick } = useDriverConnections();
  const { target, query, setTarget, setQuery } = useSearchUrlParams();
  const { targets, isPending, isError } = useSearchTargetsQuery(driver, connection);
  const run = useSearchMutation(driver, connection);
  const { open, setOpen, notes, setNotes, raise } = useSearchRaise(driver, connection);
  // an empty target means the first one the platform reports, so the page has something to run against on arrival.
  const picked = target || targets[0]?.name || "";
  const hits = run.data?.documents ?? [];

  if (!connection) {
    return (
      <Empty
        icon={SearchIcon}
        title="No instance"
        description="Register a search connection to browse what it indexes."
      />
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      {connections.length > 1 && (
        <div className="flex flex-wrap items-center gap-2">
          {connections.map((entry) => (
            <Button
              key={entry.name}
              size="sm"
              variant={entry.name === connection ? "default" : "outline"}
              onClick={() => pick(entry.name ?? "")}
            >
              {entry.name}
            </Button>
          ))}
        </div>
      )}

      <div className="grid min-h-0 flex-1 gap-4 md:grid-cols-[18rem_1fr]">
        <section className="flex min-h-0 flex-col gap-2 overflow-auto">
          <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Targets
          </h2>
          {isPending ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : isError ? (
            <p className="text-sm text-destructive">The platform did not report its targets.</p>
          ) : targets.length === 0 ? (
            <Empty
              icon={SearchIcon}
              title="No index"
              description="This platform indexes nothing yet."
            />
          ) : (
            targets.map((entry) => (
              <button
                key={entry.name}
                type="button"
                onClick={() => setTarget(entry.name ?? "")}
                className={cn(
                  "flex flex-col items-start rounded-md border px-3 py-2 text-left",
                  entry.name === picked ? "border-primary" : "border-border",
                )}
              >
                <span className="text-sm font-medium">{entry.name}</span>
                <span className="text-xs text-muted-foreground">
                  {compact(entry.documents ?? 0)} docs · {bytes(entry.sizeBytes ?? 0)}
                </span>
              </button>
            ))
          )}
        </section>

        <section className="flex min-h-0 flex-col gap-3">
          <Textarea
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={'{ "query": { "match_all": {} } }'}
            spellCheck={false}
            className="min-h-24 font-mono text-xs"
          />
          <div className="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              onClick={() => run.mutate({ op: "search", target: picked, query })}
              disabled={!picked || run.isPending}
            >
              <Play className="size-4" /> Run
            </Button>
            {picked && <span className="text-xs text-muted-foreground">on {picked}</span>}
            {run.data && <span className="text-xs text-muted-foreground">{run.data.message}</span>}
            <Button
              size="sm"
              variant="outline"
              onClick={() => setOpen(true)}
              disabled={!picked || !query.trim()}
            >
              <Trash2 className="size-4" /> Delete what this matches
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-auto rounded-md border p-3">
            {run.isError ? (
              <p className="text-sm text-destructive">{run.error.message}</p>
            ) : hits.length === 0 ? (
              <Empty
                icon={SearchIcon}
                title="No results"
                description="Run a query to see what this index holds."
              />
            ) : (
              <div className="flex flex-col gap-2">
                {hits.map((doc, index) => (
                  <div
                    key={`${index}:${doc.slice(0, 32)}`}
                    className="border-b border-border pb-2 last:border-0 last:pb-0"
                  >
                    <JsonTree value={parseHit(doc)} />
                  </div>
                ))}
              </div>
            )}
          </div>
        </section>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete what this query matches</DialogTitle>
            <DialogDescription>
              This is submitted for review, not run: the platform reports how many documents match,
              and a reviewer approves that number before anything is deleted.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <p className="text-xs text-muted-foreground">
              target <span className="font-mono text-foreground">{picked}</span>
            </p>
            <pre className="max-h-32 overflow-auto rounded-md border p-2 font-mono text-xs">
              {query}
            </pre>
            <Input
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
              placeholder="why this delete is needed"
            />
          </div>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => raise.mutate({ target: picked, query })}
              disabled={!notes.trim() || raise.isPending}
            >
              <Trash2 className="size-4" /> Raise for review
            </Button>
          </DialogFooter>
          {raise.isError && <p className="text-xs text-destructive">{raise.error.message}</p>}
          {raise.data && (
            <p className="text-xs text-muted-foreground">
              {raise.data.status === "failed" ? (
                <>Refused: {raise.data.checks?.[1]?.error ?? "the platform refused it"}</>
              ) : (
                <>
                  Raised ·{" "}
                  <Link
                    to={PAGE_ROUTES.SEARCH_REQUEST_DETAIL(driver ?? "", raise.data.id ?? "")}
                    className="underline underline-offset-4"
                  >
                    review it
                  </Link>
                </>
              )}
            </p>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
