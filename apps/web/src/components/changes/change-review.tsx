import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Link } from "react-router-dom";
import { CalendarClock, Check, Play, Undo2 } from "lucide-react";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { ChangeActivity } from "@/components/changes/change-activity";
import { ChangeSubmitDialog } from "@/components/changes/change-submit-dialog";
import { SqlEditor } from "@/components/sql/sql-editor";
import type { Change, ChangeInput, Connection, Table } from "@/lib/api";
import { PAGE_ROUTES } from "@/lib/routes";
import { cn } from "@/lib/utils";
import { P } from "@/components/ui/typography";

type ReviewOutcome =
  | { id?: string; status?: string; checks?: { target?: string; ok?: boolean; detail?: string; error?: string }[] }
  | undefined;

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  validated: "secondary",
  unverified: "outline",
  approved: "default",
  applying: "outline",
  applied: "default",
  failed: "destructive",
  rejected: "outline",
};

// ChangeDetails is the review rail: context, warnings, activity, blast radius, impact and checks.
export function ChangeDetails({ change, instance }: { change: Change; instance?: Connection }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
      {change.notes && (
        <section className="grid gap-1">
          <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">Notes</h3>
          <p className="text-sm">{change.notes}</p>
        </section>
      )}

      <ChangeAlerts change={change} />

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
        <ChangeChecks change={change} />
      </section>
    </div>
  );
}

function ChangeAlerts({ change }: { change: Change }) {
  if (change.scheduledAt && change.scheduledAt > Math.floor(Date.now() / 1000)) {
    return (
      <Alert variant="warning" title="Scheduled">
        The scheduler will apply this change at {new Date(change.scheduledAt * 1000).toLocaleString()}. Apply stays
        refused until then.
      </Alert>
    );
  }
  if (change.status === "applying") {
    return (
      <Alert variant="warning" title="Apply outcome unknown">
        {change.applyingAt
          ? `Claimed for execution at ${new Date(change.applyingAt * 1000).toLocaleString()}. `
          : "The claim time is unavailable. "}
        Inspect the target database before taking further action.
      </Alert>
    );
  }
  if (change.status === "failed") {
    return (
      <Alert variant="destructive" title="Apply failed">
        The last apply attempt failed and was recorded on this change.
      </Alert>
    );
  }
  if (change.status === "approved") {
    return (
      <Alert variant="success" title="Approved">
        Ready to apply now, or schedule it for later.
      </Alert>
    );
  }
  return null;
}

function ChangeChecks({ change }: { change: Change }) {
  if ((change.checks ?? []).length === 0) return <P>No checks recorded.</P>;
  return (change.checks ?? []).map((check) => (
    <p
      key={`${check.target}:${check.error ?? check.detail ?? ""}`}
      className={cn("text-xs", check.ok ? "text-muted-foreground" : "text-destructive")}
    >
      {check.ok ? "✓" : "✗"} <span className="font-medium text-foreground">{check.target}</span>
      {check.ok ? `: ${check.detail ?? ""}` : `: ${check.error ?? ""}`}
    </p>
  ));
}

// ChangeSqlReview is the right pane: both statements stay visible and the review actions sit below Rollback.
export function ChangeSqlReview({
  change,
  tables,
  kind,
  mayWrite,
  canApprove,
  canApply,
  canReject,
  onInverse,
  onApprove,
  onApply,
  onReject,
  onSchedule,
}: {
  change: Change;
  tables: Table[];
  kind: "dml" | "ddl";
  mayWrite: boolean;
  canApprove: boolean;
  canApply: boolean;
  canReject: boolean;
  onInverse: (input: ChangeInput) => Promise<ReviewOutcome>;
  onApprove: () => void;
  onApply: () => void;
  onReject: () => void;
  onSchedule: (at: number) => void;
}) {
  return (
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
      <ChangeActionBar
        change={change}
        kind={kind}
        mayWrite={mayWrite}
        canApprove={canApprove}
        canApply={canApply}
        canReject={canReject}
        onInverse={onInverse}
        onApprove={onApprove}
        onApply={onApply}
        onReject={onReject}
        onSchedule={onSchedule}
      />
    </div>
  );
}

function ChangeActionBar({
  change,
  kind,
  mayWrite,
  canApprove,
  canApply,
  canReject,
  onInverse,
  onApprove,
  onApply,
  onReject,
  onSchedule,
}: {
  change: Change;
  kind: "dml" | "ddl";
  mayWrite: boolean;
  canApprove: boolean;
  canApply: boolean;
  canReject: boolean;
  onInverse: (input: ChangeInput) => Promise<ReviewOutcome>;
  onApprove: () => void;
  onApply: () => void;
  onReject: () => void;
  onSchedule: (at: number) => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 border-t border-border pt-3">
      <Badge variant={STATUS_VARIANT[change.status ?? ""] ?? "outline"}>{change.status}</Badge>
      {mayWrite && (
        <ChangeSubmitDialog
          kind={kind}
          onSubmit={onInverse}
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
        <Button size="sm" variant="secondary" onClick={onApprove}>
          <Check /> Approve
        </Button>
      )}
      {canApply && (
        <>
          <ScheduleMenu change={change} onSchedule={onSchedule} />
          <Button size="sm" onClick={onApply}>
            <Play /> Apply
          </Button>
        </>
      )}
      {canReject && (
        <Button size="sm" variant="ghost" onClick={onReject}>
          <Undo2 /> Reject
        </Button>
      )}
    </div>
  );
}

const scheduleSchema = z.object({
  when: z
    .string()
    .min(1, "pick a moment")
    .refine((value) => {
      const slot = Date.parse(value);
      return !Number.isNaN(slot) && slot > Date.now();
    }, "pick a future moment"),
});
type ScheduleForm = z.infer<typeof scheduleSchema>;

// ScheduleMenu is the approved-change schedule control: Immediate clears the slot, Schedule it opens a future picker.
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
