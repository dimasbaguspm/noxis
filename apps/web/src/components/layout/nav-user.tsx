import { useNavigate } from "react-router-dom";
import { ChevronsUpDown, LogOut, Settings, ShieldCheck, UserRound } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { useAuth } from "@/app/auth-provider";
import { PAGE_ROUTES } from "@/lib/routes";

function initials(name: string, email: string): string {
  const source = name.trim() || email.trim();
  if (!source) return "?";
  const parts = source.split(/[\s.@_-]+/).filter(Boolean);
  return (parts[0]?.[0] ?? "?").concat(parts[1]?.[0] ?? "").toUpperCase();
}

// NavUser is the sidebar's bottom-left identity card and account menu: profile,
// settings, access and sign out all live here, on every page.
export function NavUser() {
  const { identity, signout } = useAuth();
  const navigate = useNavigate();
  const name = identity?.name || identity?.email || "Account";
  const email = identity?.email || identity?.sub || "";

  return (
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger render={<SidebarMenuButton size="lg" />}>
          <div className="flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg bg-sidebar-accent text-xs font-medium text-sidebar-accent-foreground">
            {initials(identity?.name ?? "", identity?.email ?? "")}
          </div>
          <div className="grid min-w-0 flex-1 text-left text-sm leading-tight">
            <span className="truncate font-medium">{name}</span>
            <span className="truncate text-xs text-muted-foreground">
              {identity?.super ? "super admin" : email || "member"}
            </span>
          </div>
          <ChevronsUpDown className="ml-auto size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" side="top" className="w-56">
          <DropdownMenuLabel>{name}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => navigate(PAGE_ROUTES.MY)}>
            <UserRound /> My profile
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate(PAGE_ROUTES.MY_SETTINGS)}>
            <Settings /> Settings
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => navigate(PAGE_ROUTES.MY_ACCESS)}>
            <ShieldCheck /> My access
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onClick={() => void signout()}>
            <LogOut /> Sign out
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  );
}
