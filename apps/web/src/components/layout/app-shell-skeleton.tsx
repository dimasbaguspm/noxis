import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuSkeleton,
  SidebarProvider,
} from "@/components/ui/sidebar";

// AppShellSkeleton preserves sidebar geometry while the session bootstraps.
export function AppShellSkeleton() {
  return (
    <SidebarProvider className="h-svh overflow-hidden">
      <Sidebar collapsible="icon" variant="inset">
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuSkeleton showIcon />
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Console</SidebarGroupLabel>
            <SidebarMenu>
              {Array.from({ length: 5 }, (_, index) => (
                <SidebarMenuItem key={index}>
                  <SidebarMenuSkeleton showIcon />
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>
      </Sidebar>
      <SidebarInset className="overflow-hidden md:border md:border-border">
        <header className="flex h-16 shrink-0 items-center gap-2 border-b border-border px-4">
          <div className="h-8 w-8 animate-pulse rounded-md bg-muted" />
          <div className="h-4 w-40 animate-pulse rounded-md bg-muted" />
        </header>
      </SidebarInset>
    </SidebarProvider>
  );
}
