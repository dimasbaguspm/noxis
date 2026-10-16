import * as React from "react";
import { PanelLeft } from "lucide-react";
import { cn } from "@/lib/utils";
import { useIsMobile } from "@/hooks/use-is-mobile";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle } from "@/components/ui/sheet";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Slot } from "@/components/ui/slot";

const SIDEBAR_WIDTH = "16rem";
const SIDEBAR_WIDTH_ICON = "3rem";

type SidebarContext = {
  state: "expanded" | "collapsed";
  open: boolean;
  setOpen: (open: boolean) => void;
  openMobile: boolean;
  setOpenMobile: (open: boolean) => void;
  isMobile: boolean;
  toggleSidebar: () => void;
};

const SidebarContext = React.createContext<SidebarContext | null>(null);

function useSidebar() {
  const context = React.useContext(SidebarContext);
  if (!context) throw new Error("useSidebar must be used inside a SidebarProvider");
  return context;
}

export function SidebarProvider({
  defaultOpen = true,
  children,
  className,
  style,
}: {
  defaultOpen?: boolean;
  children: React.ReactNode;
  className?: string;
  style?: React.CSSProperties;
}) {
  const isMobile = useIsMobile();
  const [openMobile, setOpenMobile] = React.useState(false);
  const [open, setOpen] = React.useState(defaultOpen);

  const toggleSidebar = React.useCallback(() => setOpen((v) => !v), []);

  const value = React.useMemo(
    () => ({ state: (open ? "expanded" : "collapsed") as SidebarContext["state"], open, setOpen, openMobile, setOpenMobile, isMobile, toggleSidebar }),
    [open, openMobile, isMobile, toggleSidebar],
  );

  return (
    <SidebarContext.Provider value={value}>
      <div
        data-sidebar-wrapper
        className={cn("group/sidebar-wrapper flex min-h-svh w-full", className)}
        style={
          { "--sidebar-width": SIDEBAR_WIDTH, "--sidebar-width-icon": SIDEBAR_WIDTH_ICON, ...style } as React.CSSProperties
        }
      >
        {children}
      </div>
    </SidebarContext.Provider>
  );
}

export function Sidebar({ className, side = "left", children }: { className?: string; side?: "left" | "right"; children?: React.ReactNode }) {
  const { isMobile, openMobile, setOpenMobile } = useSidebar();
  const sideClass =
    side === "right"
      ? "right-0 border-l border-l-sidebar-border"
      : "left-0 border-r border-r-sidebar-border";

  if (isMobile) {
    return (
      <Sheet open={openMobile} onOpenChange={setOpenMobile}>
        <SheetContent className={cn("w-(--sidebar-width) border-sidebar-border bg-sidebar text-sidebar-foreground", side === "right" && "right-0 left-auto")}>
          <SheetTitle className="sr-only">Sidebar</SheetTitle>
          {children}
        </SheetContent>
      </Sheet>
    );
  }

  return (
    <aside
      data-slot="sidebar"
      className={cn(
        "group-data-[sidebar-wrapper]:fixed top-0 z-10 flex h-svh w-(--sidebar-width) shrink-0 flex-col bg-sidebar text-sidebar-foreground",
        sideClass,
        "group-data-[state=collapsed]:w-(--sidebar-width-icon) group-data-[state=collapsed]:transition-[width] group-data-[state=collapsed]:duration-200 group-data-[state=collapsed]:ease-linear",
        className,
      )}
    >
      {children}
    </aside>
  );
}

export function SidebarHeader({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="sidebar-header" className={cn("flex flex-col gap-2 p-2", className)} {...props} />;
}

export function SidebarFooter({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="sidebar-footer" className={cn("flex flex-col gap-2 p-2", className)} {...props} />;
}

export function SidebarContent({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="sidebar-content" className={cn("flex min-h-0 flex-1 flex-col gap-2 overflow-auto", className)} {...props} />;
}

export function SidebarGroup({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div data-slot="sidebar-group" className={cn("relative flex w-full min-w-0 flex-col p-2", className)} {...props} />
  );
}

export function SidebarGroupLabel({
  className,
  asChild = false,
  ...props
}: React.ComponentProps<"div"> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "div";
  return (
    <Comp
      data-slot="sidebar-group-label"
      className={cn("flex h-8 shrink-0 items-center rounded-md px-2 text-xs font-medium text-muted-foreground/70 outline-none ring-sidebar-ring transition-[margin,opacity] duration-200 ease-linear group-data-[collapsible=icon]:mt-8 group-data-[collapsible=icon]:-mt-5 group-data-[collapsible=icon]:-ms-1 group-data-[collapsible=icon]:hidden", className)}
      {...(asChild ? {} : props)}
    />
  );
}

export function SidebarGroupContent({ className, ...props }: React.ComponentProps<"div">) {
  return <div data-slot="sidebar-group-content" className={cn("w-full text-sm", className)} {...props} />;
}

export function SidebarMenu({ className, ...props }: React.ComponentProps<"ul">) {
  return <ul data-slot="sidebar-menu" className={cn("flex w-full min-w-0 flex-col gap-1", className)} {...props} />;
}

export function SidebarMenuItem({ className, ...props }: React.ComponentProps<"li">) {
  return <li data-slot="sidebar-menu-item" className={cn("group/menu-item relative", className)} {...props} />;
}

const sidebarMenuButtonVariants =
  "peer/menu-button flex w-full items-center gap-2 overflow-hidden rounded-md p-2 text-left text-sm outline-none transition-[width,height,padding] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring active:bg-sidebar-accent active:text-sidebar-accent-foreground disabled:pointer-events-none disabled:opacity-50 group-has-data-[sidebar-menu-button-type=main]/group-menu-item:bg-sidebar-accent group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-2! data-[active=true]:bg-sidebar-accent data-[active=true]:font-medium data-[active=true]:text-sidebar-accent-foreground group-data-[collapsible=icon]:flex group-data-[collapsible=icon]:justify-center [&>span:last-child]:truncate";

export function SidebarMenuButton({
  asChild = false,
  isActive = false,
  size = "default",
  className,
  children,
  ...props
}: React.ComponentProps<"button"> & {
  asChild?: boolean;
  isActive?: boolean;
  size?: "default" | "sm" | "lg";
}) {
  const { isMobile, state } = useSidebar();
  const Comp = asChild ? Slot : "button";
  const sizes = {
    default: "h-9 [&>svg]:size-4 [&>svg]:shrink-0",
    sm: "h-8 gap-2 text-xs [&>svg]:size-3.5 [&>svg]:shrink-0",
    lg: "group-data-[collapsible=icon]:min-h-8! h-12 gap-2 p-2 [&>svg]:size-5 [&>svg]:shrink-0",
  } as const;
  return (
    <Tooltip>
      <TooltipTrigger render={<span className="flex min-w-0 flex-1" />}>
        <Comp
          data-slot="sidebar-menu-button"
          data-size={size}
          data-active={isActive}
          data-sidebar-menu-button-type="main"
          className={cn(sidebarMenuButtonVariants, sizes[size], className)}
          {...(asChild ? {} : props)}
        >
          {// Slot needs exactly one element child; the sr-only label only exists
          // for icon-only buttons, so asChild links keep theirs as-is.
          asChild ? (
            children
          ) : (
            <>
              {isMobile || state === "collapsed" ? (
                <span className="sr-only">{typeof children === "string" ? children : undefined}</span>
              ) : null}
              {children}
            </>
          )}
        </Comp>
      </TooltipTrigger>
      <TooltipContent side="right" hidden={isMobile || state !== "collapsed"}>
        {children}
      </TooltipContent>
    </Tooltip>
  );
}

export function SidebarMenuSub({ className, ...props }: React.ComponentProps<"ul">) {
  return (
    <ul
      data-slot="sidebar-menu-sub"
      data-sidebar-menu-sub
      className={cn(
        "mx-3.5 flex min-w-0 translate-x-px flex-col gap-1 border-l border-sidebar-border px-2.5 py-0.5 group-data-[collapsible=icon]:hidden",
        className,
      )}
      {...props}
    />
  );
}

export function SidebarMenuSubItem({ className, ...props }: React.ComponentProps<"li">) {
  return <li data-slot="sidebar-menu-sub-item" className={cn("group/sub-item relative", className)} {...props} />;
}

export function SidebarMenuSubButton({
  asChild = false,
  size = "md",
  isActive = false,
  className,
  children,
  ...props
}: React.ComponentProps<"a"> & {
  asChild?: boolean;
  size?: "sm" | "md";
  isActive?: boolean;
}) {
  const Comp = asChild ? Slot : "a";
  return (
    <Comp
      data-slot="sidebar-menu-sub-button"
      data-size={size}
      data-active={isActive}
      className={cn(
        "flex h-7 min-w-0 -translate-x-px items-center gap-2 overflow-hidden rounded-md px-2 text-sidebar-foreground/70 outline-none hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring active:bg-sidebar-accent active:text-sidebar-accent-foreground disabled:pointer-events-none disabled:opacity-50 [&>span:last-child]:truncate",
        size === "sm" && "text-xs [&>svg]:size-3",
        size === "md" && "text-sm [&>svg]:size-3.5",
        isActive && "bg-sidebar-accent text-sidebar-accent-foreground",
        className,
      )}
      {...(asChild ? {} : props)}
    >
      {children}
    </Comp>
  );
}

export function SidebarRail({ className, ...props }: React.ComponentProps<"button">) {
  const { toggleSidebar } = useSidebar();
  return (
    <button
      data-slot="sidebar-rail"
      aria-label="Toggle Sidebar"
      onClick={toggleSidebar}
      className={cn(
        "group absolute inset-y-0 z-20 hidden w-4 -translate-x-1/2 transition-all ease-linear after:absolute after:inset-y-0 after:left-1/2 after:w-[2px] hover:after:bg-sidebar-border sm:flex",
        "data-[collapsible]:w-10 data-[collapsible]:after:right-0",
        className,
      )}
      {...props}
    />
  );
}

export function SidebarTrigger({ className, onClick, ...props }: React.ComponentProps<typeof Button>) {
  const { toggleSidebar } = useSidebar();
  return (
    <Button
      data-slot="sidebar-trigger"
      variant="ghost"
      size="icon"
      onClick={(event) => {
        onClick?.(event);
        toggleSidebar();
      }}
      className={className}
      {...props}
    >
      <PanelLeft />
      <span className="sr-only">Toggle Sidebar</span>
    </Button>
  );
}

export function SidebarInset({ className, ...props }: React.ComponentProps<"main">) {
  return <main data-slot="sidebar-inset" className={cn("relative flex min-h-svh flex-1 flex-col bg-background", className)} {...props} />;
}

export function SidebarSeparator({ className, ...props }: React.ComponentProps<typeof Separator>) {
  const { isMobile } = useSidebar();
  return <Separator data-slot="sidebar-separator" className={cn("mx-2 w-auto", isMobile && "hidden", className)} {...props} />;
}

export { useSidebar, SIDEBAR_WIDTH, SIDEBAR_WIDTH_ICON };