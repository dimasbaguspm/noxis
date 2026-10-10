import { useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { ChevronRight, ChevronsUpDown, GalleryVerticalEnd, LayoutDashboard } from "lucide-react";
import { cn } from "@/lib/utils";
import { FAMILIES, type Family } from "@/lib/families";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const VERSION = "v0.1.0";

function isItemActive(pathname: string, url: string): boolean {
  if (url === "/") return pathname === "/";
  return pathname === url || pathname.startsWith(`${url}/`);
}

// SiteTitle is the dashboard-01 team-switcher pattern: the console brand and
// version open a menu that jumps straight into any engine family.
function SiteTitle() {
  const navigate = useNavigate();
  return (
    <SidebarHeader>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger render={<SidebarMenuButton size="lg" />}>
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                <GalleryVerticalEnd className="size-4" />
              </div>
              <div className="flex flex-col gap-0.5 leading-none">
                <span className="font-semibold">noXis</span>
                <span className="">{VERSION}</span>
              </div>
              <ChevronsUpDown className="ml-auto" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-56">
              {FAMILIES.map((family) => (
                <DropdownMenuItem
                  key={family.key}
                  onClick={() => navigate(family.nav[0]?.to ?? "/")}
                >
                  <family.icon className="size-4" />
                  {family.label}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarHeader>
  );
}

// FamilyMenu is the dashboard-01 nav-main item: an icon + label button that
// expands its sub-menu, open by default while one of its pages is active.
function FamilyMenu({
  family,
  pathname,
}: {
  family: Family;
  pathname: string;
}) {
  const active = family.nav.some((item) => isItemActive(pathname, item.to));
  const [open, setOpen] = useState(active);
  const Icon = family.icon;

  return (
    <SidebarMenuItem>
      <SidebarMenuButton isActive={active} onClick={() => setOpen((v) => !v)}>
        <Icon />
        <span>{family.label}</span>
        <ChevronRight
          className={cn("ml-auto transition-transform duration-200", open && "rotate-90")}
        />
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

// AppSidebar renders the dashboard-01 structure: site title on top, one
// labelled group of items whose engine families expand into sub-menus, rail.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();

  return (
    <Sidebar {...props}>
      <SiteTitle />
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Console</SidebarGroupLabel>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton asChild isActive={pathname === "/"}>
                <Link to="/" className="font-medium">
                  <LayoutDashboard />
                  <span>Instances</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
            {FAMILIES.map((family) => (
              <FamilyMenu key={family.key} family={family} pathname={pathname} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}