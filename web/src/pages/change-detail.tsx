import { Link, useNavigate, useParams } from "react-router-dom";
import { Check, Play, Undo2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
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

// ChangeDetailPage is the reviewer's focus view: one change, full context,
// its estimates and its audit trail. Changes are artifacts — there is no
// delete, only reject.
export function ChangeDetailPage({ kind }: { kind: ChangeKind }) {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const { changes, approve, apply, reject } = useChanges(kind);
  const change = changes.find((c) => c.id === id);

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

  const acted = (fn: (id: string) => Promise<unknown>) => () => void fn(id);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" onClick={() => navigate(`/${kind}`)}>
          ← {kind.toUpperCase()}
        </Button>
        <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>{change.status}</Badge>
        <span className="text-sm text-muted-foreground">{change.connection}</span>
        <span className="flex-1" />
        {(change.status === "validated" || change.status === "unverified") && (
          <Button size="sm" variant="secondary" onClick={acted(approve.run)}>
            <Check /> Approve
          </Button>
        )}
        {change.status === "approved" && (
          <Button size="sm" onClick={acted(apply.run)}>
            <Play /> Apply
          </Button>
        )}
        {change.status !== "applied" && change.status !== "rejected" && (
          <Button size="sm" variant="ghost" onClick={acted(reject.run)}>
            <Undo2 /> Reject
          </Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{change.notes}</CardTitle>
          <CardDescription>
            created {ts(change.createdAt)}
            {change.approvedAt ? ` · approved ${ts(change.approvedAt)}` : ""}
            {change.appliedAt ? ` · applied ${ts(change.appliedAt)}` : ""}
            {change.rejectedAt ? ` · rejected ${ts(change.rejectedAt)}` : ""}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <Code label="query" value={change.query ?? ""} />
          <Code label="rollback" value={change.rollback ?? ""} />
        </CardContent>
      </Card>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Estimates (dry run)</CardTitle>
            <CardDescription>measured while validating; nothing persisted</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-3 text-sm">
            <Metric label="est. duration" value={`${change.estDurationMs ?? 0} ms`} />
            <Metric label="est. rows affected" value={String(change.estAffected ?? 0)} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Applied (actual)</CardTitle>
            <CardDescription>{change.appliedAt ? "committed" : "not applied yet"}</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-3 text-sm">
            <Metric label="duration" value={`${change.appliedDurationMs ?? 0} ms`} />
            <Metric label="rows affected" value={String(change.appliedAffected ?? 0)} />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Validation trail</CardTitle>
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
          <Separator className="my-1" />
          <p className="text-xs text-muted-foreground">
            changes are immutable artifacts: rejected records stay in the list for the audit trail.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function Code({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <pre className="max-h-56 overflow-auto rounded-md border border-border bg-background p-3 font-mono text-xs">
        {value}
      </pre>
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

function ts(seconds?: number): string {
  return seconds ? new Date(seconds * 1000).toLocaleString() : "—";
}
