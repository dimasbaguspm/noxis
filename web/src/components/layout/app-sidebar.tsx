import { Link, useLocation } from "react-router-dom";
import { GalleryVerticalEnd } from "lucide-react";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarRail,
} from "@/components/ui/sidebar";
import { NAV, type NavItem } from "@/lib/nav";

const VERSION = "v0.1.0";

interface NavMainItem {
  title: string;
  url: string;
  items?: NavItem[];
}

function isItemActive(pathname: string, url: string): boolean {
  if (url === "/") return pathname === "/";
  return pathname === url || pathname.startsWith(`${url}/`);
}

// AppSidebar renders the navigation in the reference shadcn structure: a brand
// block on top, one group of nav items (families expand into sub-menus), and a
// rail. The engine families keep coming from lib/nav, so a new family only has
// to be registered there.
export function AppSidebar(props: React.ComponentProps<typeof Sidebar>) {
  const { pathname } = useLocation();
  const navMain: NavMainItem[] = NAV.map((group) => {
    if (!group.label) {
      const first = group.items[0];
      return { title: first.label, url: first.to, items: [] };
    }
    return { title: group.label, url: "#", items: group.items };
  });

  return (
    <Sidebar {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" asChild>
              <Link to="/">
                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                  <GalleryVerticalEnd className="size-4" />
                </div>
                <div className="flex flex-col gap-0.5 leading-none">
                  <span className="font-medium">noXis</span>
                  <span className="">{VERSION}</span>
                </div>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarMenu>
            {navMain.map((item) => (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton asChild>
                  {item.items?.length ? (
                    <a href={item.url} className="font-medium">{item.title}</a>
                  ) : (
                    <Link to={item.url} className="font-medium">{item.title}</Link>
                  )}
                </SidebarMenuButton>
                {item.items?.length ? (
                  <SidebarMenuSub>
                    {item.items.map((sub) => (
                      <SidebarMenuSubItem key={sub.to}>
                        <SidebarMenuSubButton asChild isActive={isItemActive(pathname, sub.to)}>
                          <Link to={sub.to}>{sub.label}</Link>
                        </SidebarMenuSubButton>
                      </SidebarMenuSubItem>
                    ))}
                  </SidebarMenuSub>
                ) : null}
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}