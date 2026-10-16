import { SidebarMenuItem } from "@/components/ui/sidebar";
import { useAuth } from "@/app/auth-provider";

function initials(name: string, email: string): string {
  const source = name.trim() || email.trim();
  if (!source) return "?";
  const parts = source.split(/[\s.@_-]+/).filter(Boolean);
  return (parts[0]?.[0] ?? "?").concat(parts[1]?.[0] ?? "").toUpperCase();
}

// NavUser is the sidebar's bottom-left identity card: avatar initials, name
// and email. Plain markup keeps it free of any dropdown or hover tooltip.
export function NavUser() {
  const { identity, authEnabled } = useAuth();
  const name = identity?.name || identity?.email || "Open mode";
  const email = identity?.email ?? (authEnabled ? "" : "authentication disabled");

  return (
    <SidebarMenuItem>
      <div className="flex items-center gap-2 rounded-md px-2 py-1">
        <div className="flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg bg-sidebar-accent text-xs font-medium text-sidebar-accent-foreground">
          {initials(identity?.name ?? "", identity?.email ?? "")}
        </div>
        <div className="grid min-w-0 flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
          <span className="truncate font-medium">{name}</span>
          <span className="truncate text-xs text-muted-foreground">{email}</span>
        </div>
      </div>
    </SidebarMenuItem>
  );
}