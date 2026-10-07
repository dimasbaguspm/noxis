import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Check, CircleHelp, FileWarning, Loader2, Play, Undo2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChangeActivity } from "@/components/changes/change-activity";
import { SqlEditor } from "@/components/sql/sql-editor";
import { useSchema } from "@/hooks/use-connection";
import { useChanges, type ChangeKind } from "@/hooks/use-data";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "secondary",
  unverified: "outline",
  approved: "default",
  applied: "default",
  failed: "destructive",
  rejected: "outline",
};

// ChangeDetailPage is the reviewer's focus view. Actions live here and only
// here: the queue stays a queue.
export function ChangeDetailPage({ kind }: { kind: ChangeKind }) {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { changes, loading, approve, apply, reject } = useChanges(kind);
  const change = changes.find((c) => c.id === id);
  const { tables } = useSchema(change?.connection);

  if (loading && !change) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Loader2 className="size-4 animate-spin" />
            Loading change {id.slice(0, 8)}
          </CardTitle>
        </CardHeader>
      </Card>
    );
  }

  if (!change) return <Unavailable kind={kind} id={id} />;

  const queue = kind === "ddl" ? PAGE_ROUTES.SQL.DDL : PAGE_ROUTES.SQL.DML;
  const act = (fn: (id: string) => Promise<unknown>) => () => void fn(id);
  const canApprove = change.status === "validated" || change.status === "unverified";
  const canApply = change.status === "approved";
  const canReject = change.status !== "applied" && change.status !== "rejected";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="ghost" size="sm" onClick={() => navigate(queue)}>
          <ArrowLeft /> {kind.toUpperCase()}
        </Button>
        <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>{change.status}</Badge>
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
              <CardDescription>Proposed {kind.toUpperCase()} change</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <Section label="Query">
                <SqlEditor value={change.query ?? ""} readOnly height="11rem" tables={tables} />
              </Section>
              <Section label="Rollback">
                <SqlEditor value={change.rollback ?? ""} readOnly height="8rem" tables={tables} />
              </Section>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-1.5 text-base">
                Validation
                <Tooltip>
                  <TooltipTrigger render={<span className="text-muted-foreground" />}>
                    <CircleHelp className="size-3.5" />
                  </TooltipTrigger>
                  <TooltipContent>
                    The dry run applies the query and then its rollback inside a transaction that is always
                    rolled back, so nothing persists.
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
              <CardTitle className="text-base">Touches</CardTitle>
              <CardDescription>Parsed from the statements</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-2 text-sm">
              <Identifiers label="tables" values={change.affectedTables} />
              <Identifiers label="columns" values={change.affectedColumns} variant="outline" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Impact</CardTitle>
              <CardDescription>Dry run estimates, apply actuals</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3">
              <div className="grid grid-cols-2 gap-3">
                <Metric label="est. duration" value={`${change.estDurationMs ?? 0} ms`} />
                <Metric label="est. rows" value={`~${change.estAffected ?? 0}`} />
                <Metric label="duration" value={change.appliedAt ? `${change.appliedDurationMs ?? 0} ms` : "—"} />
                <Metric label="rows affected" value={change.appliedAt ? String(change.appliedAffected ?? 0) : "—"} />
              </div>
              <Separator />
              <p className="text-xs text-muted-foreground">
                Rejected changes stay in the queue as artifacts. Nothing is deleted.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

// Unavailable is the state for an id that is not in the queue: explained,
// with the way out, instead of a bare "not found".
function Unavailable({ kind, id }: { kind: ChangeKind; id: string }) {
  const queue = kind === "ddl" ? PAGE_ROUTES.SQL.DDL : PAGE_ROUTES.SQL.DML;
  return (
    <Card className="mx-auto max-w-2xl">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <FileWarning className="size-4 text-amber-500" />
          Change {id.slice(0, 8)} is not available to review
        </CardTitle>
        <CardDescription>
          The queue has no record with this id. Nothing is broken, this is usually one of these:
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <ul className="flex flex-col gap-2 text-sm text-muted-foreground">
          <li className="flex gap-2">
            <span className="text-foreground">1.</span>
            Validation failed when it was submitted, so it was never recorded. Failed proposals are
            returned to the submitter with the reason and are not kept.
          </li>
          <li className="flex gap-2">
            <span className="text-foreground">2.</span>
            The link came from another browser or session. The queue is per noXis instance, not per user.
          </li>
          <li className="flex gap-2">
            <span className="text-foreground">3.</span>
            The change belongs to a different connection that has since been removed, which also removes
            its changes.
          </li>
        </ul>
        <div className="flex flex-wrap gap-2">
          <Button render={<Link to={queue} />} size="sm">
            Back to {kind.toUpperCase()} queue
          </Button>
          <Button variant="outline" size="sm" render={<Link to={PAGE_ROUTES.SQL.QUERY} />}>
            Go to Query
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid gap-1.5">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      {children}
    </div>
  );
}

function Identifiers({
  label,
  values,
  variant = "secondary",
}: {
  label: string;
  values?: string[];
  variant?: "secondary" | "outline";
}) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="mt-0.5 flex flex-wrap gap-1">
        {(values ?? []).length === 0 ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          (values ?? []).map((value) => (
            <Badge key={value} variant={variant}>
              {value}
            </Badge>
          ))
        )}
      </div>
    </div>
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
