import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronRight, KeyRound, LayoutDashboard, UsersRound } from "lucide-react";
import { LogoMark } from "@/components/brand/logo-mark";
import { cn } from "@/lib/utils";
import { sections, type Section } from "@/lib/console-nav";
import { useDriversQuery } from "@/hooks/use-config";
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
  useSidebar,
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

// SectionMenu expands the sidebar first when a collapsed rail hides its children.
function SectionMenu({ section, pathname }: { section: Section; pathname: string }) {
  const active = section.nav.some((item) => isItemActive(pathname, item.to));
  const [open, setOpen] = useState(active);
  const { state, isMobile, setOpen: setSidebarOpen } = useSidebar();
  const Icon = section.icon;

  function toggle() {
    if (state === "collapsed" && !isMobile) {
      setSidebarOpen(true);
      setOpen(true);
      return;
    }
    setOpen((value) => !value);
  }

  return (
    <SidebarMenuItem>
      <SidebarMenuButton isActive={active} tooltip={section.label} onClick={toggle}>
        <Icon />
        <span>{section.label}</span>
        <ChevronRight
          className={cn(
            "ml-auto transition-transform duration-200 group-data-[collapsible=icon]:hidden",
            open && "rotate-90",
          )}
        />
      </SidebarMenuButton>
      {open && (
        <SidebarMenuSub>
          {section.nav.map((item) => (
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
      )}
    </SidebarMenuItem>
  );
}

// AppSidebar renders the static title and grouped instance/engine navigation. The engine groups come from the driver
// catalog, so a driver the console has never seen still appears under the capability its module declares.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const { isAdmin } = useAuth();
  const { data: drivers } = useDriversQuery();

  return (
    <Sidebar collapsible="icon" {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Console</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              <InstancesMenu pathname={pathname} />
              {sections(drivers).map((section) => (
                <SectionMenu key={section.key} section={section} pathname={pathname} />
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
