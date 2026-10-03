import { CircleDot, CircleSlash, Clock, Play, ShieldCheck, Undo2 } from "lucide-react";
import type { Change } from "@/lib/api";

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
    { label: "Created", at: change.createdAt, icon: CircleDot, tone: "text-muted-foreground" },
    {
      label: "Validated",
      at: change.createdAt,
      icon: ShieldCheck,
      tone: "text-sky-500",
      hint: `${change.checks?.length ?? 0} check(s) recorded`,
    },
  ];
  if (change.rejectedAt) {
    steps.push({ label: "Rejected", at: change.rejectedAt, icon: CircleSlash, tone: "text-destructive" });
  }
  if (change.approvedAt) {
    steps.push({ label: "Approved", at: change.approvedAt, icon: ShieldCheck, tone: "text-amber-500" });
  }
  if (change.appliedAt) {
    steps.push({
      label: "Executed",
      at: change.appliedAt,
      icon: Play,
      tone: "text-emerald-500",
      hint: `${change.appliedDurationMs ?? 0} ms · ${change.appliedAffected ?? 0} row(s) affected`,
    });
  }
  if (!change.appliedAt && change.status !== "rejected") {
    steps.push({
      label: "Scheduled",
      icon: Clock,
      tone: "text-muted-foreground",
      hint: "runs on approve + apply (no scheduler yet)",
    });
  }
  if (change.status === "failed") {
    steps.push({ label: "Failed", at: change.appliedAt ?? change.createdAt, icon: Undo2, tone: "text-destructive" });
  }

  return (
    <ol className="relative ml-3 border-l border-border">
      {steps.map((step, i) => (
        <li key={`${step.label}-${i}`} className="mb-4 ml-4 last:mb-0">
          <span className="absolute -left-[7px] flex size-3.5 items-center justify-center rounded-full border border-border bg-background">
            <step.icon className={`size-2.5 ${step.tone}`} />
          </span>
          <p className="text-sm font-medium">{step.label}</p>
          <p className="text-xs text-muted-foreground">
            {step.at ? new Date(step.at * 1000).toLocaleString() : "—"}
            {step.hint ? ` · ${step.hint}` : ""}
          </p>
        </li>
      ))}
    </ol>
  );
}
