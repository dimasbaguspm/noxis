import { useParams } from "react-router-dom";
import { Check, Play, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Empty } from "@/components/ui/empty";
import { useAuth } from "@/app/auth-provider";
import { useSearchChangeAction, useSearchChangeQuery } from "@/hooks/use-search-changes";
import { compact } from "@/lib/format";

// requestOf reads the stored request back out of the change: the query is the platform's own document, so it is shown
// as written rather than re-rendered into a shape the console would have to understand.
function requestOf(stored?: string): { target: string; query: string } {
  try {
    const parsed = JSON.parse(stored ?? "{}") as { target?: string; query?: string };
    return { target: parsed.target ?? "", query: parsed.query ?? "" };
  } catch {
    return { target: "", query: stored ?? "" };
  }
}

// SearchRequestDetailPage is the review of one search change: what it targets, the query that decides it, the impact the
// platform counted, and the decision itself.
export default function SearchRequestDetailPage() {
  const { driver = "", id = "" } = useParams();
  const detail = useSearchChangeQuery(driver, id);
  const act = useSearchChangeAction(driver);
  const { canAccess } = useAuth();

  if (detail.isPending) return <p className="text-sm text-muted-foreground">Loading…</p>;
  const change = detail.data;
  if (!change) {
    return (
      <Empty
        icon={X}
        title="Change not found"
        description="It may have been rejected and removed, or it belongs to another driver."
      />
    );
  }

  const mayWrite = canAccess(change.connection ?? "", "write");
  const status = change.status ?? "";
  const { target, query } = requestOf(change.query);
  const canApprove = mayWrite && (status === "validated" || status === "unverified");
  const canApply = mayWrite && status === "approved";
  const canReject =
    mayWrite && status !== "applied" && status !== "rejected" && status !== "applying";

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="secondary">{status || "unknown"}</Badge>
        <span className="text-sm font-medium">{change.notes}</span>
        {change.estAffected ? (
          <span className="text-xs text-muted-foreground">
            {compact(change.estAffected)} event(s) affected
          </span>
        ) : null}
      </div>

      <section className="flex flex-col gap-1">
        <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Target
        </h2>
        <p className="font-mono text-sm">{target || change.affectedTables?.[0] || "—"}</p>
      </section>

      <section className="flex flex-col gap-1">
        <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Query</h2>
        <pre className="overflow-auto rounded-md border p-3 font-mono text-xs">{query}</pre>
      </section>

      <section className="flex flex-col gap-1">
        <h2 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Checks
        </h2>
        {(change.checks ?? []).length === 0 ? (
          <p className="text-xs text-muted-foreground">No checks recorded.</p>
        ) : (
          (change.checks ?? []).map((check) => (
            <p
              key={`${check.target}:${check.error ?? check.detail ?? ""}`}
              className="text-xs text-muted-foreground"
            >
              {check.ok ? "✓" : "✗"}{" "}
              <span className="font-medium text-foreground">{check.target}</span>
              {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
            </p>
          ))
        )}
      </section>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          onClick={() => act.mutate({ id, action: "approve" })}
          disabled={!canApprove || act.isPending}
        >
          <Check className="size-4" /> Approve
        </Button>
        <Button
          size="sm"
          onClick={() => act.mutate({ id, action: "apply" })}
          disabled={!canApply || act.isPending}
        >
          <Play className="size-4" /> Apply
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => act.mutate({ id, action: "reject" })}
          disabled={!canReject || act.isPending}
        >
          <X className="size-4" /> Reject
        </Button>
        {act.isError && <span className="text-xs text-destructive">{act.error.message}</span>}
      </div>
    </div>
  );
}
