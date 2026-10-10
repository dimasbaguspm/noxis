import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ChevronRight, GalleryVerticalEnd, LayoutDashboard, ShieldCheck } from "lucide-react";
import { cn } from "@/lib/utils";
import { FAMILIES, type Family } from "@/lib/families";
import { usePagedInstancesQuery } from "@/hooks/use-instances";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
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
          <SidebarMenuButton size="lg" asChild>
            <Link to="/">
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <GalleryVerticalEnd className="size-4" />
              </div>
              <div className="flex flex-col gap-0.5 leading-none">
                <span className="font-semibold">noXis</span>
                <span className="">{VERSION}</span>
              </div>
            </Link>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>
  );
}

// InstancesMenu lists every registered instance from the cached query, with
// the dashboard first; open by default while on the dashboard or a detail.
function InstancesMenu({ pathname }: { pathname: string }) {
  const { data } = usePagedInstancesQuery();
  const instances = data?.instances ?? [];
  const onDetail = pathname.startsWith("/instances/");
  const active = pathname === "/" || onDetail;
  const [open, setOpen] = useState(active);

  return (
    <SidebarMenuItem>
      <SidebarMenuButton isActive={active} onClick={() => setOpen((v) => !v)}>
        <LayoutDashboard />
        <span>Instances</span>
        <ChevronRight className={cn("ml-auto transition-transform duration-200", open && "rotate-90")} />
      </SidebarMenuButton>
      {open && (
        <SidebarMenuSub>
          <SidebarMenuSubItem>
            <SidebarMenuSubButton asChild isActive={pathname === "/"}>
              <Link to="/">
                <span>All instances</span>
              </Link>
            </SidebarMenuSubButton>
          </SidebarMenuSubItem>
          {instances.map((instance) => (
            <SidebarMenuSubItem key={instance.id}>
              <SidebarMenuSubButton
                asChild
                isActive={isItemActive(pathname, `/instances/${instance.id}`)}
              >
                <Link to={`/instances/${instance.id}`}>
                  <span>{instance.name}</span>
                </Link>
              </SidebarMenuSubButton>
            </SidebarMenuSubItem>
          ))}
        </SidebarMenuSub>
      )}
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

  return (
    <Sidebar {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Console</SidebarGroupLabel>
          <SidebarMenu>
            <InstancesMenu pathname={pathname} />
            {FAMILIES.map((family) => (
              <FamilyMenu key={family.key} family={family} pathname={pathname} />
            ))}
            <SidebarMenuItem>
              <SidebarMenuButton asChild isActive={pathname === "/settings"}>
                <Link to="/settings">
                  <ShieldCheck />
                  <span>Access</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}