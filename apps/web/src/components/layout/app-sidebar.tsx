import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronRight, KeyRound, LayoutDashboard, UsersRound } from "lucide-react";
import { LogoMark } from "@/components/brand/logo-mark";
import { cn } from "@/lib/utils";
import { FAMILIES, type Family } from "@/lib/families";
import { PAGE_ROUTES } from "@/lib/routes";
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
          <SidebarMenuButton render={<Link to={PAGE_ROUTES.INSTANCES} />} size="lg" className="data-[slot=sidebar-menu-button]:p-1.5!">
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
      <SidebarMenuButton render={<Link to={PAGE_ROUTES.INSTANCES} />} isActive={active} tooltip="Instances">
        <LayoutDashboard />
        <span>Instances</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

// FamilyMenu expands the sidebar first when a collapsed rail hides its children.
function FamilyMenu({ family, pathname }: { family: Family; pathname: string }) {
  const active = family.nav.some((item) => isItemActive(pathname, item.to));
  const [open, setOpen] = useState(active);
  const { state, isMobile, setOpen: setSidebarOpen } = useSidebar();
  const Icon = family.icon;

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
      <SidebarMenuButton isActive={active} tooltip={family.label} onClick={toggle}>
        <Icon />
        <span>{family.label}</span>
        <ChevronRight className={cn("ml-auto transition-transform duration-200 group-data-[collapsible=icon]:hidden", open && "rotate-90")} />
      </SidebarMenuButton>
      {open && (
        <SidebarMenuSub>
          {family.nav.map((item) => (
            <SidebarMenuSubItem key={item.to}>
              <SidebarMenuSubButton render={<Link to={item.to} />} isActive={isItemActive(pathname, item.to)}>
                <span>{item.label}</span>
              </SidebarMenuSubButton>
            </SidebarMenuSubItem>
          ))}
        </SidebarMenuSub>
      )}
    </SidebarMenuItem>
  );
}

// AppSidebar renders the static title and grouped instance/engine navigation.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const { isAdmin } = useAuth();

  return (
    <Sidebar collapsible="icon" {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Console</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              <InstancesMenu pathname={pathname} />
              {FAMILIES.map((family) => (
                <FamilyMenu key={family.key} family={family} pathname={pathname} />
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
                  <SidebarMenuButton render={<Link to={PAGE_ROUTES.USERS} />} isActive={pathname.startsWith(PAGE_ROUTES.USERS)} tooltip="Users">
                    <UsersRound />
                    <span>Users</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton render={<Link to={PAGE_ROUTES.ACCESS} />} isActive={pathname.startsWith(PAGE_ROUTES.ACCESS)} tooltip="Access">
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
