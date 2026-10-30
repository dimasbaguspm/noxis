import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

// QueuePanel is the left tile every queue shares: the filters scroll, the actions stay at the foot of the tile so they are
// always in the same place, whatever the filter list grows to.
export function QueuePanel({
  children,
  actions,
  className,
}: {
  children: ReactNode;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex min-h-0 flex-1 flex-col gap-4", className)}>
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">{children}</div>
      {actions ? <div className="flex shrink-0 flex-col gap-2">{actions}</div> : null}
    </div>
  );
}
