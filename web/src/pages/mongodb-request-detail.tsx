import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Check, CircleHelp, FileWarning, Loader2, Play, Undo2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { JsonEditor } from "@/components/sql/json-editor";
import { ChangeActivity } from "@/components/changes/change-activity";
import { useApp } from "@/app/app-provider";
import { useMongoChanges } from "@/hooks/use-mongodb";
import type { MongoRequest } from "@/lib/api";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

// MongoRequestDetailPage is the review of one request: what it would change,
// what the dry run proved, and the trail. There is no rollback editor because
// no rollback exists for a document change.
export default function MongoRequestDetailPage() {
  const { id = "" } = useParams();
  const { can } = useApp();
  const { changes, loading, approve, apply, reject } = useMongoChanges();
  const change = changes.find((c) => c.id === id);

  if (loading && !change) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Loader2 className="size-4 animate-spin" />
            Loading request {id.slice(0, 8)}
          </CardTitle>
        </CardHeader>
      </Card>
    );
  }
  if (!change) return <Unavailable id={id} />;

  const request = parseRequest(change.query);
  const act = (fn: (id: string) => Promise<unknown>) => () => void fn(id);
  const mayReview = can(change.connection, "admin");
  const canApprove = mayReview && (change.status === "validated" || change.status === "unverified");
  const canApply = mayReview && change.status === "approved";
  const canReject = mayReview && change.status !== "applied" && change.status !== "rejected";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="ghost" size="sm" render={<Link to={PAGE_ROUTES.MONGODB.REQUESTS} />}>
          <ArrowLeft /> Requests
        </Button>
        <Badge variant={statusVariant(change.status)}>{change.status}</Badge>
        <span className="font-mono text-xs text-muted-foreground">{change.id?.slice(0, 8)}</span>
        <span className="text-sm text-muted-foreground">{change.connection}</span>
        <span className="flex-1" />
        {canApprove && (
          <Button size="sm" variant="secondary" onClick={act(approve.run)}>
            <Check /> Approve
          </Button>
        )}
        {canApply && (
          <Button size="sm" onClick={act(apply.run)}>
            <Play /> Apply
          </Button>
        )}
        {canReject && (
          <Button size="sm" variant="ghost" onClick={act(reject.run)}>
            <Undo2 /> Reject
          </Button>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{change.notes}</CardTitle>
              <CardDescription>
                {request ? `${request.op} on ${request.database}.${request.collection}` : "request unreadable"}
                {request?.many ? " (many)" : ""}
              </CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <div className="grid gap-1.5">
                <p className="text-xs font-medium text-muted-foreground">Filter</p>
                <JsonEditor value={request?.filter ?? ""} readOnly height="8rem" />
              </div>
              {request?.op !== "delete" && (
                <div className="grid gap-1.5">
                  <p className="text-xs font-medium text-muted-foreground">Value</p>
                  <JsonEditor value={request?.value ?? ""} readOnly height="8rem" />
                </div>
              )}
              <div className="rounded-md border border-amber-500/30 bg-amber-500/5 p-3 text-xs text-amber-500">
                No rollback: a document change cannot be undone by the request itself. The dry run below is what
                the review stands on.
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                Verification
                <Tooltip>
                  <TooltipTrigger render={<span className="text-muted-foreground" />}>
                    <CircleHelp className="size-3.5" />
                  </TooltipTrigger>
                  <TooltipContent>
                    The matching documents were copied into a scratch collection, the write ran there, and the
                    scratch database was dropped. It proves the filter, the value and the affected count; unique
                    index conflicts only appear on apply.
                  </TooltipContent>
                </Tooltip>
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-1.5">
              {(change.checks ?? []).map((check) => (
                <p
                  key={`${check.target}:${check.error ?? check.detail ?? ""}`}
                  className={cn("text-sm", check.ok ? "text-muted-foreground" : "text-destructive")}
                >
                  {check.ok ? "✓" : "✗"} <span className="font-medium text-foreground">{check.target}</span>
                  {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
                </p>
              ))}
            </CardContent>
          </Card>
        </div>

        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Activity</CardTitle>
            </CardHeader>
            <CardContent>
              <ChangeActivity change={change} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Target</CardTitle>
              <CardDescription>parsed from the request</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-2 text-sm">
              <div className="flex flex-wrap gap-1">
                {(change.affectedTables ?? []).map((table) => (
                  <Badge key={table} variant="secondary">
                    {table}
                  </Badge>
                ))}
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Impact</CardTitle>
              <CardDescription>dry run estimate, apply actual</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3">
              <div className="grid grid-cols-2 gap-3">
                <Metric label="est. duration" value={`${change.estDurationMs ?? 0} ms`} />
                <Metric label="est. documents" value={`~${change.estAffected ?? 0}`} />
                <Metric label="duration" value={change.appliedAt ? `${change.appliedDurationMs ?? 0} ms` : "—"} />
                <Metric
                  label="documents affected"
                  value={change.appliedAt ? String(change.appliedAffected ?? 0) : "—"}
                />
              </div>
              <Separator />
              <p className="text-xs text-muted-foreground">
                Rejected requests stay in the queue as artifacts. Nothing is deleted.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

// parseRequest reads the request back out of the change: the console stores it
// in the query column as JSON.
function parseRequest(query?: string): MongoRequest | undefined {
  if (!query) return undefined;
  try {
    return JSON.parse(query) as MongoRequest;
  } catch {
    return undefined;
  }
}

function Unavailable({ id }: { id: string }) {
  return (
    <Card className="mx-auto max-w-2xl">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <FileWarning className="size-4 text-amber-500" />
          Request {id.slice(0, 8)} is not available to review
        </CardTitle>
        <CardDescription>
          The queue has no MongoDB request with this id. Usually one of these:
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <ul className="flex flex-col gap-2 text-sm text-muted-foreground">
          <li className="flex gap-2">
            <span className="text-foreground">1.</span>
            The dry run failed when it was submitted, so it was never recorded. Failed proposals are returned to
            the submitter with the reason.
          </li>
          <li className="flex gap-2">
            <span className="text-foreground">2.</span>
            The link came from another session: the queue is per noXis instance, not per user.
          </li>
          <li className="flex gap-2">
            <span className="text-foreground">3.</span>
            It belongs to a connection that was removed, which also removes its requests.
          </li>
        </ul>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" render={<Link to={PAGE_ROUTES.MONGODB.REQUESTS} />}>
            Back to Requests
          </Button>
          <Button size="sm" variant="outline" render={<Link to={PAGE_ROUTES.MONGODB.REQUEST_NEW} />}>
            New request
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="font-medium tabular-nums">{value}</p>
    </div>
  );
}

function statusVariant(status?: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "validated":
      return "secondary";
    case "approved":
    case "applied":
      return "default";
    case "failed":
      return "destructive";
  }
  return "outline";
}
