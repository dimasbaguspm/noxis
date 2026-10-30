import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Check, CircleHelp, FileWarning, Loader2, Play, Undo2 } from "lucide-react";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { JsonEditor } from "@/components/sql/json-editor";
import { ChangeActivity } from "@/components/changes/change-activity";
import { Workspace } from "@/components/layout/workspace";
import { useAuth } from "@/app/auth-provider";
import {
  useMongoApproveChangeMutation,
  useMongoApplyChangeMutation,
  useMongoChangeGetQuery,
  useMongoRejectChangeMutation,
} from "@/hooks/use-mongodb";
import type { MongoRequest } from "@/lib/api";
import { PAGE_ROUTES } from "@/constants/routes";
import { useCrumbLabel } from "@/app/crumb-provider";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

// MongoRequestDetailPage shows the reviewed write, dry-run checks, and activity; Mongo writes have no rollback.
export default function MongoRequestDetailPage() {
  const { id = "" } = useParams();
  const { isInstanceAdmin } = useAuth();
  const { data: change, isPending: loading } = useMongoChangeGetQuery(id);
  const approve = useMongoApproveChangeMutation();
  const apply = useMongoApplyChangeMutation();
  const reject = useMongoRejectChangeMutation();
  // the trail names the request instead of its id once the envelope lands
  useCrumbLabel(
    PAGE_ROUTES.REQUEST_DETAIL(id),
    change?.notes || change?.affectedTables?.join(", "),
  );

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
  const act = (fn: (id: string) => void) => () => fn(id);
  const mayReview = isInstanceAdmin(change.connection);
  const canApprove = mayReview && (change.status === "validated" || change.status === "unverified");
  const canApply = mayReview && change.status === "approved";
  const canReject =
    mayReview &&
    change.status !== "applied" &&
    change.status !== "rejected" &&
    change.status !== "applying";

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={statusVariant(change.status)}>{change.status}</Badge>
              <span className="text-sm text-muted-foreground">{change.connection}</span>
            </div>
            <p className="text-sm">{change.notes}</p>

            {change.status === "applying" && (
              <Alert variant="warning" title="Apply outcome unknown">
                {change.applyingAt
                  ? `Claimed for execution at ${new Date(change.applyingAt * 1000).toLocaleString()}. `
                  : "The claim time is unavailable. "}
                Inspect the target database before taking further action.
              </Alert>
            )}

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Target
              </h3>
              <div className="flex flex-wrap gap-1">
                {(change.affectedTables ?? []).map((table) => (
                  <Badge key={table} variant="secondary">
                    {table}
                  </Badge>
                ))}
                {(change.affectedTables ?? []).length === 0 && (
                  <span className="text-xs text-muted-foreground">—</span>
                )}
              </div>
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Activity
              </h3>
              <ChangeActivity change={change} />
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Impact
              </h3>
              <div className="grid grid-cols-2 gap-3">
                <Metric label="Est. duration" value={`${change.estDurationMs ?? 0} ms`} />
                <Metric label="Est. documents" value={`~${change.estAffected ?? 0}`} />
                <Metric
                  label="Duration"
                  value={change.appliedAt ? `${change.appliedDurationMs ?? 0} ms` : "—"}
                />
                <Metric
                  label="Documents affected"
                  value={change.appliedAt ? String(change.appliedAffected ?? 0) : "—"}
                />
              </div>
            </section>
          </div>
        }
        right={
          <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-auto">
            <div className="grid gap-1">
              <P className="font-medium">Filter</P>
              <JsonEditor value={request?.filter ?? ""} readOnly height="7rem" />
            </div>
            {request?.op !== "delete" && (
              <div className="grid gap-1">
                <P className="font-medium">Value</P>
                <JsonEditor value={request?.value ?? ""} readOnly height="7rem" />
              </div>
            )}
            <div className="rounded-md border border-amber-500/30 bg-amber-500/5 p-3 text-xs text-amber-500">
              No rollback: a document change cannot be undone by the request itself. The dry run is
              what the review stands on.
            </div>

            <section className="grid gap-1">
              <h3 className="flex items-center gap-1.5 text-xs font-medium tracking-wide text-muted-foreground uppercase">
                Verification
                <Tooltip>
                  <TooltipTrigger render={<span className="text-muted-foreground" />}>
                    <CircleHelp className="size-3.5" />
                  </TooltipTrigger>
                  <TooltipContent>
                    The matching documents were copied into a scratch collection, the write ran
                    there, and the scratch database was dropped. It proves the filter, the value and
                    the affected count; unique index conflicts only appear on apply.
                  </TooltipContent>
                </Tooltip>
              </h3>
              {(change.checks ?? []).map((check) => (
                <p
                  key={`${check.target}:${check.error ?? check.detail ?? ""}`}
                  className={cn("text-xs", check.ok ? "text-muted-foreground" : "text-destructive")}
                >
                  {check.ok ? "✓" : "✗"}{" "}
                  <span className="font-medium text-foreground">{check.target}</span>
                  {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
                </p>
              ))}
            </section>

            <div className="mt-auto flex flex-wrap items-center gap-2 border-t border-border pt-3">
              <Button variant="ghost" size="sm" render={<Link to={PAGE_ROUTES.REQUESTS} />}>
                <ArrowLeft /> Requests
              </Button>
              <span className="flex-1" />
              {canApprove && (
                <Button size="sm" variant="secondary" onClick={act(approve.mutate)}>
                  <Check /> Approve
                </Button>
              )}
              {canApply && (
                <Button size="sm" onClick={act(apply.mutate)}>
                  <Play /> Apply
                </Button>
              )}
              {canReject && (
                <Button size="sm" variant="ghost" onClick={act(reject.mutate)}>
                  <Undo2 /> Reject
                </Button>
              )}
            </div>
          </div>
        }
      />
    </div>
  );
}

// parseRequest reads the request back out of the change: the console stores it in the query column as JSON.
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
            The dry run failed when it was submitted, so it was never recorded. Failed proposals are
            returned to the submitter with the reason.
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
          <Button size="sm" render={<Link to={PAGE_ROUTES.REQUESTS} />}>
            Back to Requests
          </Button>
          <Button size="sm" variant="outline" render={<Link to={PAGE_ROUTES.REQUEST_NEW} />}>
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
      <P>{label}</P>
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
    case "applying":
      return "outline";
  }
  return "outline";
}
