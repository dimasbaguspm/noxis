import { Link, useLocation } from "react-router-dom";
import { KeyRound, LayoutDashboard, UsersRound } from "lucide-react";
import { LogoMark } from "@/components/brand/logo-mark";
import { pagesOf, type NavItem } from "@/lib/console-nav";
import { useDriversQuery } from "@/hooks/use-config";
import { PAGE_ROUTES } from "@/constants/routes";
import { NavUser } from "@/components/layout/nav-user";
import { DriverPicker } from "@/components/layout/driver-picker";
import { useAuth } from "@/app/auth-provider";
import { useDriver } from "@/hooks/use-driver";
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

// PageMenu is one engine page of the selected driver: a flat link, because the driver already scopes it.
function PageMenu({ item, pathname }: { item: NavItem; pathname: string }) {
  const Icon = item.icon;
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<Link to={item.to} />}
        isActive={isItemActive(pathname, item.to)}
        tooltip={item.label}
      >
        <Icon />
        <span>{item.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

// AppSidebar renders the static title, the instance list, the driver picker and the pages that driver supports.
// The pages come from the driver catalog, so a driver the console has never seen still gets the pages its module serves.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const { isAdmin } = useAuth();
  const { data: drivers = [] } = useDriversQuery();
  const { driver } = useDriver();
  const pages = pagesOf(drivers, driver);

  return (
    <Sidebar collapsible="icon" {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Console</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              <InstancesMenu pathname={pathname} />
              {drivers.length > 0 && <DriverPicker driver={driver} drivers={drivers} />}
              {pages.map((item) => (
                <PageMenu key={item.to} item={item} pathname={pathname} />
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
