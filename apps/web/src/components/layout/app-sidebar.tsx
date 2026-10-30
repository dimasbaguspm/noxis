import { Link, useLocation } from "react-router-dom";
import { KeyRound, LayoutDashboard, UsersRound } from "lucide-react";
import { LogoMark } from "@/components/brand/logo-mark";
import { DriverMark } from "@/components/brand/driver-mark";
import { pagesOf, type NavItem } from "@/lib/console-nav";
import { useDriversQuery } from "@/hooks/use-config";
import { countLabel, useInstanceCountsQuery } from "@/hooks/use-instance-counts";
import { PAGE_ROUTES } from "@/constants/routes";
import { NavUser } from "@/components/layout/nav-user";
import { useAuth } from "@/app/auth-provider";
import type { DriverInfo } from "@/lib/api";
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

// DriverMenu is one driver: a row that carries its pages beneath it, and the same row collapsed to its monogram.
// The monogram is a span, so it is exempted from the rule that sends the row's text to screen readers in the rail.
function DriverMenu({
  entry,
  pages,
  count,
  pathname,
}: {
  entry: DriverInfo;
  pages: NavItem[];
  count?: number;
  pathname: string;
}) {
  const first = pages[0];
  const active = pages.some((page) => isItemActive(pathname, page.to));
  const label = countLabel(count);

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={first ? <Link to={first.to} /> : undefined}
        isActive={active}
        tooltip={entry.label}
        className="group-data-[collapsible=icon]:[&>span:first-child]:not-sr-only group-data-[collapsible=icon]:[&>span:first-child]:size-5!"
      >
        <DriverMark driver={entry.driver} />
        <span className="truncate">{entry.label}</span>
        {label && (
          <span className="ml-auto text-xs tabular-nums text-muted-foreground">{label}</span>
        )}
      </SidebarMenuButton>
      <SidebarMenuSub>
        {pages.map((item) => (
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
  );
}

// AppSidebar renders the instance list and every driver the catalog declares, in one unlabelled list, then the admin
// section. A driver is a row with its pages beneath it, because a menu that hides the fleet is a menu nobody can read.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const { isAdmin } = useAuth();
  const { data: drivers = [] } = useDriversQuery();
  const counts = useInstanceCountsQuery().data?.counts ?? {};

  return (
    <Sidebar collapsible="icon" variant="inset" {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              <InstancesMenu pathname={pathname} />
              {drivers.map((entry) => (
                <DriverMenu
                  key={entry.driver}
                  entry={entry}
                  pages={pagesOf(drivers, entry.driver)}
                  count={counts[entry.driver]}
                  pathname={pathname}
                />
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
