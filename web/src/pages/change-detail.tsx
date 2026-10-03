import { Link, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, Check, Play, Undo2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { ChangeActivity } from "@/components/changes/change-activity";
import { SqlEditor } from "@/components/sql/sql-editor";
import { useSchema } from "@/hooks/use-connection";
import { useChanges, type ChangeKind } from "@/hooks/use-data";
import { cn } from "@/lib/utils";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "secondary",
  unverified: "outline",
  approved: "default",
  applied: "default",
  failed: "destructive",
  rejected: "outline",
};

// ChangeDetailPage is the reviewer's focus view: the change, its estimates vs
// actuals, its activity trail and the only place actions live.
export function ChangeDetailPage({ kind }: { kind: ChangeKind }) {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { changes, approve, apply, reject } = useChanges(kind);
  const change = changes.find((c) => c.id === id);
  const { tables } = useSchema(change?.connection);

  if (!change) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Change not found</CardTitle>
          <CardDescription>
            <Link to={`/${kind}`} className="underline">
              back to {kind.toUpperCase()}
            </Link>
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const act = (fn: (id: string) => Promise<unknown>) => () => void fn(id);
  const canApprove = change.status === "validated" || change.status === "unverified";
  const canApply = change.status === "approved";
  const canReject = change.status !== "applied" && change.status !== "rejected";

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <Button variant="ghost" size="sm" onClick={() => navigate(`/${kind}`)}>
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
              <CardDescription>proposed {kind.toUpperCase()} change</CardDescription>
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
              <CardTitle className="text-base">Validation trail</CardTitle>
              <CardDescription>the dry run proves the rollback undoes the change; nothing persists</CardDescription>
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
              {change.status === "failed" && (
                <p className="text-xs text-destructive">
                  validation failed — submit a new change for the corrected statements
                </p>
              )}
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
              <CardTitle className="text-base">Impact</CardTitle>
              <CardDescription>estimates from the dry run, actuals from apply</CardDescription>
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
                changes are artifacts: rejected records stay in the list, nothing is deleted.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
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

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="font-medium tabular-nums">{value}</p>
    </div>
  );
}
