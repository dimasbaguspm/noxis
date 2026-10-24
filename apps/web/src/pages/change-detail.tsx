import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link, useParams } from "react-router-dom";
import { CalendarClock, Check, FileWarning, Loader2, Play, Undo2 } from "lucide-react";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { ChangeActivity } from "@/components/changes/change-activity";
import { ChangeSubmitDialog } from "@/components/changes/change-submit-dialog";
import { SqlEditor } from "@/components/sql/sql-editor";
import { Workspace } from "@/components/layout/workspace";
import { useAuth } from "@/app/auth-provider";
import { useConnectionSchemaQuery } from "@/hooks/use-sql";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import {
  useApproveChangeMutation,
  useApplyChangeMutation,
  useChangeGetQuery,
  useRejectChangeMutation,
  useScheduleChangeMutation,
  useSubmitChangeMutation,
  type ChangeKind,
} from "@/hooks/use-changes";
import type { Change } from "@/lib/api";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "secondary",
  unverified: "outline",
  approved: "default",
  applying: "outline",
  applied: "default",
  failed: "destructive",
  rejected: "outline",
};

// ChangeDetailPage is the reviewer's focus view. Actions live here and only here: the queue stays a queue.
export function ChangeDetailPage({ kind }: { kind: ChangeKind }) {
  const { id = "" } = useParams();
  const { canAccess, isInstanceAdmin } = useAuth();
  const { data: instancesData } = usePagedInstancesQuery();
  const detail = useChangeGetQuery(id);
  const change = detail.data;
  const gone = detail.isError;
  const fetching = detail.isPending;
  const { data: tables } = useConnectionSchemaQuery(change?.connection);
  const approve = useApproveChangeMutation();
  const apply = useApplyChangeMutation();
  const reject = useRejectChangeMutation();
  const submit = useSubmitChangeMutation();
  const schedule = useScheduleChangeMutation();

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

  const instance = (instancesData?.instances ?? []).find((i) => i.name === change.connection);
  const act = (fn: (id: string) => void) => () => fn(id);
  const mayReview = isInstanceAdmin(change.connection);
  const mayWrite = canAccess(change.connection, "write");
  const canApprove = mayReview && (change.status === "validated" || change.status === "unverified");
  const canApply = mayReview && change.status === "approved";
  const canReject = mayReview && change.status !== "applied" && change.status !== "rejected" && change.status !== "applying";

  return (
    <div className="flex h-full min-h-0">
      <Workspace
        leftLabel="Change details"
        rightLabel="Change SQL"
        left={
          <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
            {change.notes && (
              <section className="grid gap-1">
                <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Notes</h3>
                <p className="text-sm">{change.notes}</p>
              </section>
            )}

            {change.scheduledAt && change.scheduledAt > Math.floor(Date.now() / 1000) ? (
              <Alert variant="warning" title="Scheduled">
                The scheduler will apply this change at{" "}
                {new Date(change.scheduledAt * 1000).toLocaleString()}. Apply stays refused until then.
              </Alert>
            ) : change.status === "applying" ? (
              <Alert variant="warning" title="Apply outcome unknown">
                {change.applyingAt ? `Claimed for execution at ${new Date(change.applyingAt * 1000).toLocaleString()}. ` : "The claim time is unavailable. "}Inspect the target database before taking further action.
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

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Instance</h3>
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-medium">{change.connection}</span>
                {instance?.driver && <Badge variant="secondary">{instance.driver}</Badge>}
              </div>
              {instance ? (
                <>
                  <p className="font-mono text-xs text-muted-foreground">
                    {instance.driver === "sqlite"
                      ? instance.database
                      : `${instance.host}:${instance.port} · ${instance.database}`}
                  </p>
                  <Button size="xs" variant="outline" className="w-fit" render={<Link to={PAGE_ROUTES.INSTANCE_DETAIL(instance.id ?? "")} />}>
                    Open instance
                  </Button>
                </>
              ) : (
                <p className="text-xs text-muted-foreground">Instance details are unavailable.</p>
              )}
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Activity</h3>
              <ChangeActivity change={change} />
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Touches</h3>
              <Identifiers label="Tables" values={change.affectedTables} />
              <Identifiers label="Columns" values={change.affectedColumns} variant="outline" />
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Impact</h3>
              <div className="grid grid-cols-2 gap-3">
                <Metric label="Est. duration" value={`${change.estDurationMs ?? 0} ms`} />
                <Metric label="Est. rows" value={`~${change.estAffected ?? 0}`} />
                <Metric label="Duration" value={change.appliedAt ? `${change.appliedDurationMs ?? 0} ms` : "—"} />
                <Metric label="Rows affected" value={change.appliedAt ? String(change.appliedAffected ?? 0) : "—"} />
              </div>
            </section>

            <section className="grid gap-2">
              <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Checks</h3>
              {(change.checks ?? []).length === 0 ? (
                <P>No checks recorded.</P>
              ) : (
                (change.checks ?? []).map((check) => (
                  <p
                    key={`${check.target}:${check.error ?? check.detail ?? ""}`}
                    className={cn("text-xs", check.ok ? "text-muted-foreground" : "text-destructive")}
                  >
                    {check.ok ? "✓" : "✗"} <span className="font-medium text-foreground">{check.target}</span>
                    {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
                  </p>
                ))
              )}
            </section>
          </div>
        }
        right={
          <div className="flex min-h-0 flex-1 flex-col gap-3">
            <div className="flex min-h-0 flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-muted-foreground">Original</span>
              <div className="min-h-0 flex-1">
                <SqlEditor value={change.query ?? ""} readOnly height="100%" tables={tables} />
              </div>
            </div>
            <div className="flex min-h-0 flex-1 flex-col gap-1">
              <span className="text-xs font-medium text-muted-foreground">Rollback</span>
              <div className="min-h-0 flex-1">
                <SqlEditor value={change.rollback ?? ""} readOnly height="100%" tables={tables} />
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2 border-t border-border pt-3">
              <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>{change.status}</Badge>
              {mayWrite && (
                <ChangeSubmitDialog
                  kind={kind}
                  onSubmit={(input) => submit.mutateAsync(input)}
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
              <span className="flex-1" />
              {canApprove && (
                <Button size="sm" variant="secondary" onClick={act(approve.mutate)}>
                  <Check /> Approve
                </Button>
              )}
              {canApply && (
                <>
                  <ScheduleMenu
                    change={change}
                    onSchedule={(at) => void schedule.mutate({ id, scheduledAt: at })}
                  />
                  <Button size="sm" onClick={act(apply.mutate)}>
                    <Play /> Apply
                  </Button>
                </>
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

const scheduleSchema = z.object({
  when: z
    .string()
    .min(1, "pick a moment")
    .refine((v) => {
      const slot = Date.parse(v);
      return !Number.isNaN(slot) && slot > Date.now();
    }, "pick a future moment"),
});
type ScheduleForm = z.infer<typeof scheduleSchema>;

// ScheduleMenu is the approved-change schedule control: Immediate clears the slot, Schedule it opens a small picker for a future time.
function ScheduleMenu({ change, onSchedule }: { change: Change; onSchedule: (at: number) => void }) {
  const [open, setOpen] = useState(false);
  const scheduled = (change.scheduledAt ?? 0) > 0;

  const {
    register,
    handleSubmit,
    reset: resetForm,
    formState: { errors },
  } = useForm<ScheduleForm>({
    defaultValues: { when: "" },
    resolver: zodResolver(scheduleSchema),
  });
  const onValid = handleSubmit((values) => {
    onSchedule(Math.floor(Date.parse(values.when) / 1000));
    setOpen(false);
  });

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
              resetForm({ when: "" });
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
          <SheetBody>
            <form id="schedule-change" onSubmit={onValid}>
              <Field>
                <FieldLabel htmlFor="schedule-at">Run at</FieldLabel>
                <Input id="schedule-at" type="datetime-local" aria-invalid={!!errors.when} {...register("when")} />
                <FieldDescription>a future moment; the scheduler applies the change once.</FieldDescription>
                <FieldError errors={[errors.when]} />
              </Field>
            </form>
          </SheetBody>
          <SheetFooter>
            <Button size="sm" type="submit" form="schedule-change">
              Schedule
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </>
  );
}

// Unavailable is the state for an id that is not in the queue: the shared empty state, centered, with the way out.
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
      <P>{label}</P>
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
      <P>{label}</P>
      <p className="font-medium tabular-nums">{value}</p>
    </div>
  );
}
