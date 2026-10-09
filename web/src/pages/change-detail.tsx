import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { title } from "radash";
import { ArrowLeft, CalendarClock, Check, CircleHelp, FileWarning, Loader2, Play, Undo2 } from "lucide-react";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty } from "@/components/ui/empty";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetBody,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ChangeActivity } from "@/components/changes/change-activity";
import { ChangeSubmitDialog } from "@/components/changes/change-submit-dialog";
import { SqlEditor } from "@/components/sql/sql-editor";
import { useAuth } from "@/app/auth-provider";
import { useSchema } from "@/hooks/use-connection";
import { useChanges, type ChangeKind } from "@/hooks/use-data";
import { api, type Change } from "@/lib/api";
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
  const { can } = useAuth();
  const { approve, apply, reject, submit, schedule } = useChanges(kind);
  // the queues are paginated, so a detail page reads its change by id
  const [change, setChange] = useState<Change>();
  const [gone, setGone] = useState(false);
  const [fetching, setFetching] = useState(true);
  const { tables } = useSchema(change?.connection);

  const refresh = useCallback(async () => {
    setFetching(true);
    try {
      setChange(await api.changes.get(id));
      setGone(false);
    } catch {
      setGone(true);
    } finally {
      setFetching(false);
    }
  }, [id]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  if (fetching && !change) {
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

  if (gone || !change) return <Unavailable kind={kind} id={id} />;

  const queue = kind === "ddl" ? PAGE_ROUTES.SQL.DDL : PAGE_ROUTES.SQL.DML;
  const act = (fn: (id: string) => Promise<unknown>) => () =>
    void fn(id).finally(() => void refresh());
  const mayReview = can(change.connection, "admin");
  const mayWrite = can(change.connection, "write");
  const canApprove = mayReview && (change.status === "validated" || change.status === "unverified");
  const canApply = mayReview && change.status === "approved";
  const canReject = mayReview && change.status !== "applied" && change.status !== "rejected";

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
          <>
            <ScheduleMenu
              change={change}
              onSchedule={(at) => void schedule.run(id, at).finally(() => void refresh())}
            />
            <Button size="sm" onClick={act(apply.run)}>
              <Play /> Apply
            </Button>
          </>
        )}
        {canReject && (
          <Button size="sm" variant="ghost" onClick={act(reject.run)}>
            <Undo2 /> Reject
          </Button>
        )}
      </div>

      {change.scheduledAt && change.scheduledAt > Math.floor(Date.now() / 1000) ? (
        <Alert variant="warning" title="Scheduled">
          The scheduler will apply this change at{" "}
          {new Date(change.scheduledAt * 1000).toLocaleString()}. Apply stays refused until then.
        </Alert>
      ) : change.status === "failed" ? (
        <Alert variant="destructive" title="Apply failed">
          The last apply attempt failed and was recorded on this change.
        </Alert>
      ) : change.status === "approved" ? (
        <Alert variant="success" title="Approved">
          Ready to apply now, or schedule it for later.
        </Alert>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-[1fr_320px]">
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">{change.notes}</CardTitle>
              <CardDescription>Proposed {kind.toUpperCase()} change</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <Section label={title("original query")}>
                <SqlEditor value={change.query ?? ""} readOnly height="11rem" tables={tables} />
              </Section>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-xs text-muted-foreground">
                  Prefills this rollback as the new original query
                </span>
                {mayWrite && (
                  <ChangeSubmitDialog
                    kind={kind}
                    onSubmit={(input) => submit.run(input)}
                    initial={{
                      connection: change.connection,
                      notes: change.notes ? `[Rollback] ${change.notes}` : "[Rollback]",
                      query: change.rollback ?? "",
                      rollback: change.query ?? "",
                    }}
                    trigger={
                      <Button size="xs" variant="outline">
                        <Undo2 /> Raise inverse change
                      </Button>
                    }
                  />
                )}
              </div>
              <Section label={title("rollback query")}>
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
              <Identifiers label={title("tables")} values={change.affectedTables} />
              <Identifiers label={title("columns")} values={change.affectedColumns} variant="outline" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Impact</CardTitle>
              <CardDescription>Dry run estimates, apply actuals</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3">
              <div className="grid grid-cols-2 gap-3">
                <Metric label={title("est. duration")} value={`${change.estDurationMs ?? 0} ms`} />
                <Metric label={title("est. rows")} value={`~${change.estAffected ?? 0}`} />
                <Metric label={title("duration")} value={change.appliedAt ? `${change.appliedDurationMs ?? 0} ms` : "—"} />
                <Metric label={title("rows affected")} value={change.appliedAt ? String(change.appliedAffected ?? 0) : "—"} />
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

// ScheduleMenu is the approved-change schedule control: Immediate clears the
// slot, Schedule it opens a small picker for a future time.
function ScheduleMenu({ change, onSchedule }: { change: Change; onSchedule: (at: number) => void }) {
  const [open, setOpen] = useState(false);
  const [when, setWhen] = useState("");
  const slot = when ? Math.floor(new Date(when).getTime() / 1000) : 0;
  const valid = slot > Math.floor(Date.now() / 1000);
  const scheduled = (change.scheduledAt ?? 0) > 0;

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button size="sm" variant="outline" />}>
          <CalendarClock /> {scheduled ? "Scheduled" : "Schedule"}
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={() => onSchedule(0)}>Immediate</DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => {
              setWhen("");
              setOpen(true);
            }}
          >
            Schedule it
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent className="sm:max-w-sm">
          <SheetHeader>
            <SheetTitle>Schedule change</SheetTitle>
            <SheetDescription>Pick when the scheduler should apply this approved change.</SheetDescription>
          </SheetHeader>
          <SheetBody className="grid gap-1.5">
            <Label htmlFor="schedule-at">Run at</Label>
            <Input
              id="schedule-at"
              type="datetime-local"
              value={when}
              onChange={(e) => setWhen(e.target.value)}
            />
          </SheetBody>
          <SheetFooter>
            <Button
              size="sm"
              disabled={!valid}
              onClick={() => {
                onSchedule(slot);
                setOpen(false);
              }}
            >
              Schedule
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  );
}

// Unavailable is the state for an id that is not in the queue: the shared
// empty state, centered, with the way out.
function Unavailable({ kind, id }: { kind: ChangeKind; id: string }) {
  const queue = kind === "ddl" ? PAGE_ROUTES.SQL.DDL : PAGE_ROUTES.SQL.DML;
  return (
    <Empty
      icon={FileWarning}
      title={`Change ${id.slice(0, 8)} is not available`}
      description="The queue has no record with this id. It may have failed validation, belong to a removed connection, or come from another session."
      action={
        <div className="flex flex-wrap justify-center gap-2">
          <Button size="sm" render={<Link to={queue} />}>
            Back to {kind.toUpperCase()} queue
          </Button>
          <Button size="sm" variant="outline" render={<Link to={PAGE_ROUTES.SQL.QUERY} />}>
            Go to Query
          </Button>
        </div>
      }
    />
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
