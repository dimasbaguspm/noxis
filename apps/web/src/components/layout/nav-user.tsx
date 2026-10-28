import { Link } from "react-router-dom";
import { ChevronRight } from "lucide-react";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { useAuth } from "@/app/auth-provider";
import { PAGE_ROUTES } from "@/constants/routes";

function initials(name: string, email: string): string {
  const source = name.trim() || email.trim();
  if (!source) return "?";
  const parts = source.split(/[\s.@_-]+/).filter(Boolean);
  return (parts[0]?.[0] ?? "?").concat(parts[1]?.[0] ?? "").toUpperCase();
}

// NavUser is the sidebar's account row: the whole row opens /my, where profile, settings and access live.
export function NavUser() {
  const { identity } = useAuth();
  const name = identity?.name || identity?.email || "Account";
  const email = identity?.email || identity?.sub || "";

  return (
    <SidebarMenuItem>
      <SidebarMenuButton render={<Link to={PAGE_ROUTES.MY} />} size="lg" tooltip="My account">
        <div className="flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg bg-sidebar-accent text-xs font-medium text-sidebar-accent-foreground">
          {initials(identity?.name ?? "", identity?.email ?? "")}
        </div>
        <div className="grid min-w-0 flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
          <span className="truncate font-medium">{name}</span>
          <span className="truncate text-xs text-muted-foreground">
            {identity?.role === "super"
              ? "super admin"
              : identity?.role === "admin"
                ? "admin"
                : email || "member"}
          </span>
        </div>
        <ChevronRight className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
