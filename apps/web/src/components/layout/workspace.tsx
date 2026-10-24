import type { ReactNode } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { cn } from "@/lib/utils";

// Workspace is the shared split surface: a filter/nav rail on the left and the content card on the right.
// Both cards stay mounted, so loading and empty states never collapse the layout.
export function Workspace({ left, right, leftLabel, rightLabel }: {
  left: ReactNode
  right: ReactNode
  leftLabel: string
  rightLabel: string
}) {
  return (
    <ResizablePanelGroup orientation="horizontal" className="min-h-0 min-w-0 flex-1">
      <ResizablePanel defaultSize={20} minSize={16}>
        <WorkspacePane label={leftLabel} className="mr-2">
          {left}
        </WorkspacePane>
      </ResizablePanel>
      <ResizableHandle withHandle />
      <ResizablePanel defaultSize={80} minSize={40} className="min-w-0">
        <WorkspacePane label={rightLabel} className="ml-2">
          {right}
        </WorkspacePane>
      </ResizablePanel>
    </ResizablePanelGroup>
  );
}

// WorkspacePane is the card body both sides share; pages provide their own scrolling and empty state.
export function WorkspacePane({ children, label, className }: {
  children: ReactNode
  label: string
  className?: string
}) {
  return (
    <Card aria-label={label} className={cn("flex h-full min-h-0 flex-col", className)}>
      <CardContent className="flex min-h-0 flex-1 flex-col">{children}</CardContent>
    </Card>
  );
}
