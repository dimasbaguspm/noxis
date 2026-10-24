import type { ComponentType, ReactNode } from "react";
import { AlertTriangle, CheckCircle2, Clock, Info } from "lucide-react";
import { cn } from "@/lib/utils";

// Alert is a tinted notice for a notable state: approved, scheduled, failed, and so on.
type Variant = "default" | "info" | "success" | "warning" | "destructive";

const ICONS: Record<Variant, ComponentType<{ className?: string }>> = {
  default: Info,
  info: Info,
  success: CheckCircle2,
  warning: Clock,
  destructive: AlertTriangle,
};

const TONES: Record<Variant, string> = {
  default: "border-border text-foreground",
  info: "border-sky-500/40 bg-sky-500/5 text-sky-600 dark:text-sky-400",
  success: "border-emerald-500/40 bg-emerald-500/5 text-emerald-600 dark:text-emerald-400",
  warning: "border-amber-500/40 bg-amber-500/5 text-amber-600 dark:text-amber-400",
  destructive: "border-destructive/40 bg-destructive/5 text-destructive",
};

export function Alert({
  variant = "default",
  title,
  children,
  className,
}: {
  variant?: Variant;
  title?: string;
  children?: ReactNode;
  className?: string;
}) {
  const Icon = ICONS[variant];
  return (
    <div
      role="alert"
      className={cn(
        "flex items-start gap-2 rounded-md border px-3 py-2 text-sm",
        TONES[variant],
        className,
      )}
    >
      <Icon className="mt-0.5 size-4 shrink-0" />
      <div className="min-w-0">
        {title && <p className="font-medium leading-snug">{title}</p>}
        {children && <div className="text-xs opacity-90">{children}</div>}
      </div>
    </div>
  );
}
