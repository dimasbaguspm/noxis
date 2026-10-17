import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronRight, GalleryVerticalEnd, LayoutDashboard, Settings, UserRound, UsersRound } from "lucide-react";
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
} from "@/components/ui/sidebar";

const VERSION = "v0.1.0";

function isItemActive(pathname: string, url: string): boolean {
  if (url === "/") return pathname === "/";
  return pathname === url || pathname.startsWith(`${url}/`);
}

// SiteTitle is the static brand: icon tile, product name and version. No
// switcher — the console has one identity, whatever engine family you open.
function SiteTitle() {
  return (
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            asChild
            size="lg"
            className="data-[slot=sidebar-menu-button]:p-1.5!"
          >
            <Link to="/">
              <GalleryVerticalEnd className="size-5!" />
              <span className="text-base font-semibold">noXis</span>
              <span className="ml-auto text-xs text-muted-foreground">{VERSION}</span>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>
  );
}

// InstancesMenu is a plain link to the instance list; the fleet is big enough
// that an expanding submenu only adds a level nobody asks for.
function InstancesMenu({ pathname }: { pathname: string }) {
  const active = pathname === "/" || pathname.startsWith("/instances/");
  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild isActive={active}>
        <Link to="/">
          <LayoutDashboard />
          <span>Instances</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

// FamilyMenu is the dashboard-01 nav-main item: an icon + label button that
// expands its sub-menu, open by default while one of its pages is active.
function FamilyMenu({ family, pathname }: { family: Family; pathname: string }) {
  const active = family.nav.some((item) => isItemActive(pathname, item.to));
  const [open, setOpen] = useState(active);
  const Icon = family.icon;

  return (
    <SidebarMenuItem>
      <SidebarMenuButton isActive={active} onClick={() => setOpen((v) => !v)}>
        <Icon />
        <span>{family.label}</span>
        <ChevronRight className={cn("ml-auto transition-transform duration-200", open && "rotate-90")} />
      </SidebarMenuButton>
      {open && (
        <SidebarMenuSub>
          {family.nav.map((item) => (
            <SidebarMenuSubItem key={item.to}>
              <SidebarMenuSubButton asChild isActive={isItemActive(pathname, item.to)}>
                <Link to={item.to}>
                  <span>{item.label}</span>
                </Link>
              </SidebarMenuSubButton>
            </SidebarMenuSubItem>
          ))}
        </SidebarMenuSub>
      )}
    </SidebarMenuItem>
  );
}

// AppSidebar renders the dashboard-01 structure: static site title on top,
// one labelled group of items — the Instances submenu first, then each engine
// family expanding into its sub-menus — and a rail.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const { isSuper } = useAuth();

  return (
    <Sidebar {...props}>
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
              {isSuper && (
                <SidebarMenuItem>
                  <SidebarMenuButton asChild isActive={pathname.startsWith(PAGE_ROUTES.USERS)}>
                    <Link to={PAGE_ROUTES.USERS}>
                      <UsersRound />
                      <span>Users</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup>
          <SidebarGroupLabel>Account</SidebarGroupLabel>
          <SidebarGroupContent className="flex flex-col gap-2">
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={pathname === PAGE_ROUTES.PROFILE}>
                  <Link to={PAGE_ROUTES.PROFILE}>
                    <UserRound />
                    <span>Profile</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={pathname === PAGE_ROUTES.SETTINGS}>
                  <Link to={PAGE_ROUTES.SETTINGS}>
                    <Settings />
                    <span>Settings</span>
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}