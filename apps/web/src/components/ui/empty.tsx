import type { ComponentType, ReactNode } from "react";
import { cn } from "@/lib/utils";
import { H2, P } from "@/components/ui/typography";

// Empty is a shared no-content state with optional explanation and action.
export function Empty({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon?: ComponentType<{ className?: string }>;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-2 px-6 py-12 text-center",
        className,
      )}
    >
      {Icon && (
        <span className="flex size-10 items-center justify-center rounded-full bg-accent text-muted-foreground">
          <Icon className="size-5" />
        </span>
      )}
      <H2>{title}</H2>
      {description && <P className="max-w-md">{description}</P>}
      {action && <div className="mt-2">{action}</div>}
    </div>
  );
}
