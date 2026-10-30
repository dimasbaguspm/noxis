import { Link, useLocation } from "react-router-dom";
import { KeyRound, LayoutDashboard, UsersRound } from "lucide-react";
import { LogoMark } from "@/components/brand/logo-mark";
import { DriverMark } from "@/components/brand/driver-mark";
import { pagesOf } from "@/lib/console-nav";
import { useDriversQuery } from "@/hooks/use-config";
import { countLabel, useInstanceCountsQuery } from "@/hooks/use-instance-counts";
import { PAGE_ROUTES } from "@/constants/routes";
import { NavUser } from "@/components/layout/nav-user";
import { useAuth } from "@/app/auth-provider";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar";

const VERSION = "v0.1.0";

function isItemActive(pathname: string, url: string): boolean {
  return pathname === url || pathname.startsWith(`${url}/`);
}

// SiteTitle is the static brand: icon tile, product name and version.
function SiteTitle() {
  return (
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            render={<Link to={PAGE_ROUTES.INSTANCES} />}
            size="lg"
            className="data-[slot=sidebar-menu-button]:p-1.5!"
          >
            <LogoMark className="size-5 shrink-0" />
            <span className="text-base font-semibold">noXis</span>
            <span className="ml-auto text-xs text-muted-foreground">{VERSION}</span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>
  );
}

// InstancesMenu is a plain link to the instance list; the fleet is big enough that an expanding submenu only adds a level nobody asks for.
function InstancesMenu({ pathname }: { pathname: string }) {
  const active = isItemActive(pathname, PAGE_ROUTES.INSTANCES);
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<Link to={PAGE_ROUTES.INSTANCES} />}
        isActive={active}
        tooltip="Instances"
      >
        <LayoutDashboard />
        <span>Instances</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

// AppSidebar renders the instance list, every driver the catalog declares with its pages always open, and the admin
// section. A driver is a label with its count and its pages beneath it, because a menu that hides the fleet is a menu
// nobody can read.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const { isAdmin } = useAuth();
  const { data: drivers = [] } = useDriversQuery();
  const counts = useInstanceCountsQuery().data?.counts ?? {};

  return (
    <Sidebar collapsible="icon" {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Console</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              <InstancesMenu pathname={pathname} />
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        {/* the rail has no room for a label row, so the driver list is only drawn when the sidebar is open */}
        <SidebarGroup className="group-data-[collapsible=icon]:hidden">
          <SidebarGroupLabel>Drivers</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {drivers.map((entry) => (
                <SidebarMenuItem key={entry.driver}>
                  <div className="flex items-center gap-2 px-2 py-1.5 text-sm">
                    <DriverMark driver={entry.driver} />
                    <span className="truncate">{entry.label}</span>
                    {countLabel(counts[entry.driver]) && (
                      <span className="ml-auto text-xs tabular-nums text-muted-foreground">
                        {countLabel(counts[entry.driver])}
                      </span>
                    )}
                  </div>
                  <SidebarMenuSub>
                    {pagesOf(drivers, entry.driver).map((item) => (
                      <SidebarMenuSubItem key={item.to}>
                        <SidebarMenuSubButton
                          render={<Link to={item.to} />}
                          isActive={isItemActive(pathname, item.to)}
                        >
                          <span>{item.label}</span>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    ))}
                  </SidebarMenuSub>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        {isAdmin && (
          <SidebarGroup>
            <SidebarGroupLabel>Administration</SidebarGroupLabel>
            <SidebarGroupContent className="flex flex-col gap-2">
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    render={<Link to={PAGE_ROUTES.USERS} />}
                    isActive={pathname.startsWith(PAGE_ROUTES.USERS)}
                    tooltip="Users"
                  >
                    <UsersRound />
                    <span>Users</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    render={<Link to={PAGE_ROUTES.ACCESS} />}
                    isActive={pathname.startsWith(PAGE_ROUTES.ACCESS)}
                    tooltip="Access"
                  >
                    <KeyRound />
                    <span>Access</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}
