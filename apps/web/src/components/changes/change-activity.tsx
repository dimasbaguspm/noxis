import { CircleDot, CircleSlash, Clock, Play, ShieldCheck, Undo2 } from "lucide-react";
import type { Change } from "@/lib/api";
import { H2, P } from "@/components/ui/typography";

type Step = {
  label: string;
  at?: number;
  icon: typeof CircleDot;
  tone: string;
  hint?: string;
};

// ChangeActivity renders the audit trail: what happened, when, and by which
// status — the timeline a reviewer needs.
export function ChangeActivity({ change }: { change: Change }) {
  const steps: Step[] = [
    {
      label: "Created",
      at: change.createdAt,
      icon: CircleDot,
      tone: "text-muted-foreground",
      hint: change.createdBy ? `by ${change.createdBy}` : undefined,
    },
    {
      label: "Validated",
      at: change.createdAt,
      icon: ShieldCheck,
      tone: "text-sky-500",
      hint: `${change.checks?.length ?? 0} check(s) recorded`,
    },
  ];
  if (change.rejectedAt) {
    steps.push({
      label: "Rejected",
      at: change.rejectedAt,
      icon: CircleSlash,
      tone: "text-destructive",
      hint: change.rejectedBy ? `by ${change.rejectedBy}` : undefined,
    });
  }
  if (change.approvedAt) {
    steps.push({
      label: "Approved",
      at: change.approvedAt,
      icon: ShieldCheck,
      tone: "text-amber-500",
      hint: change.approvedBy ? `by ${change.approvedBy}` : undefined,
    });
  }
  if (change.scheduledAt || change.scheduledBy) {
    steps.push({
      label: change.scheduledAt ? "Scheduled" : "Schedule cleared",
      at: change.scheduledAt,
      icon: Clock,
      tone: "text-muted-foreground",
      hint: [
        change.scheduledBy && `by ${change.scheduledBy}`,
        change.scheduledAt && (change.appliedAt ? "ran on the scheduled tick" : "runs on the next tick"),
      ]
        .filter(Boolean)
        .join(" · "),
    });
  }
  if (change.status === "applying") {
    steps.push({
      label: "Applying",
      at: change.applyingAt,
      icon: Clock,
      tone: "text-amber-500",
      hint: change.appliedBy ? `claimed by ${change.appliedBy}; outcome may be unknown` : "outcome may be unknown",
    });
  }
  if (change.appliedAt) {
    steps.push({
      label: "Executed",
      at: change.appliedAt,
      icon: Play,
      tone: "text-emerald-500",
      hint: [change.appliedBy && `by ${change.appliedBy}`, `${change.appliedDurationMs ?? 0} ms · ${change.appliedAffected ?? 0} row(s) affected`]
        .filter(Boolean)
        .join(" · "),
    });
  }
  if (change.status === "failed") {
    steps.push({
      label: "Failed",
      at: change.applyingAt ?? change.createdAt,
      icon: Undo2,
      tone: "text-destructive",
      hint: change.appliedBy ? `attempted by ${change.appliedBy}` : undefined,
    });
  }

  return (
    <ol className="relative ml-3 border-l border-border">
      {steps.map((step, i) => (
        <li key={`${step.label}-${i}`} className="mb-4 ml-4 last:mb-0">
          <span className="absolute -left-[7px] flex size-3.5 items-center justify-center rounded-full border border-border bg-background">
            <step.icon className={`size-2.5 ${step.tone}`} />
          </span>
          <H2>{step.label}</H2>
          <P>
            {step.at ? new Date(step.at * 1000).toLocaleString() : "—"}
            {step.hint ? ` · ${step.hint}` : ""}
          </P>
        </li>
      ))}
    </ol>
  );
}
