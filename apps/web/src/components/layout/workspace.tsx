import { Children, Fragment, createContext, isValidElement, useContext, type ComponentProps, type ReactNode } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable";
import { cn } from "@/lib/utils";

type WorkspaceOrientation = "horizontal" | "vertical";

type WorkspaceContextValue = {
  orientation: WorkspaceOrientation;
  padded: boolean;
  scroll: boolean;
};

const WorkspaceContext = createContext<WorkspaceContextValue>({
  orientation: "horizontal",
  padded: true,
  scroll: false,
});

// useWorkspace exposes the defaults a pane inherits from its nearest split.
export function useWorkspace() {
  return useContext(WorkspaceContext);
}

// withHandles keeps the composition declarative: panes stay direct children and the split inserts one handle between each pair.
function withHandles(children: ReactNode) {
  const items = Children.toArray(children).filter((child) => child !== null && child !== undefined);
  return items.map((child, index) => (
    <Fragment key={isValidElement(child) && child.key != null ? String(child.key) : `pane-${index}`}>
      {index > 0 && <ResizableHandle withHandle />}
      {child}
    </Fragment>
  ));
}

type WorkspacePaneProps = ComponentProps<typeof ResizablePanel> & {
  padded?: boolean;
  scroll?: boolean;
};

// WorkspacePane is the card body a tile shares. Padding and scrolling are opt-out so editors and graphs can fill the tile.
export function WorkspacePane({ children, className, padded, scroll, ...props }: WorkspacePaneProps) {
  const inherited = useWorkspace();
  const contentPadding = padded ?? inherited.padded;
  const contentScroll = scroll ?? inherited.scroll;
  return (
    <ResizablePanel className={className} {...props}>
      <Card className={cn("flex h-full min-h-0 flex-col", !contentPadding && "py-0")}>
        <CardContent
          className={cn(
            "flex min-h-0 flex-1 flex-col",
            !contentPadding && "px-0",
            contentScroll && "overflow-y-auto",
          )}
        >
          {children}
        </CardContent>
      </Card>
    </ResizablePanel>
  );
}

type WorkspaceSplitProps = {
  children: ReactNode;
  direction?: WorkspaceOrientation;
  defaultSize?: number;
  minSize?: number;
  maxSize?: number;
  className?: string;
  groupClassName?: string;
};

// WorkspaceSplit is one nested resize group. Its children stay cards; the split itself is only the panel around them.
export function WorkspaceSplit({
  children,
  direction = "vertical",
  defaultSize,
  minSize,
  maxSize,
  className,
  groupClassName,
}: WorkspaceSplitProps) {
  const inherited = useWorkspace();
  return (
    <ResizablePanel defaultSize={defaultSize} minSize={minSize} maxSize={maxSize} className={cn("min-h-0 min-w-0", className)}>
      <WorkspaceContext.Provider value={{ ...inherited, orientation: direction }}>
        <ResizablePanelGroup
          orientation={direction}
          className={cn("h-full min-h-0 min-w-0 gap-2", groupClassName)}
        >
          {withHandles(children)}
        </ResizablePanelGroup>
      </WorkspaceContext.Provider>
    </ResizablePanel>
  );
}

type WorkspaceProps = {
  left?: ReactNode;
  right?: ReactNode;
  children?: ReactNode;
  className?: string;
};

// Workspace is the shared split surface. The legacy left/right signature stays as a thin wrapper while pages migrate to the compound API.
function WorkspaceRoot({ left, right, children, className }: WorkspaceProps) {
  const content =
    children ??
    (left !== undefined || right !== undefined
      ? [
          <WorkspacePane key="left" defaultSize={17} minSize={14}>
            {left}
          </WorkspacePane>,
          <WorkspacePane key="right" defaultSize={83} minSize={40}>
            {right}
          </WorkspacePane>,
        ]
      : undefined);

  return (
    <WorkspaceContext.Provider value={{ orientation: "horizontal", padded: true, scroll: false }}>
      <ResizablePanelGroup
        orientation="horizontal"
        className={cn("min-h-0 min-w-0 flex-1 gap-2", className)}
      >
        {withHandles(content)}
      </ResizablePanelGroup>
    </WorkspaceContext.Provider>
  );
}

export const Workspace = Object.assign(WorkspaceRoot, {
  Pane: WorkspacePane,
  Split: WorkspaceSplit,
  Context: WorkspaceContext,
  useWorkspace,
});
