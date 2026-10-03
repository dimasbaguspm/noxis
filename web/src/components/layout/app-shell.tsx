import { NavLink, Outlet } from "react-router-dom";
import { Activity, BarChart3, Database, LayoutDashboard, PencilRuler } from "lucide-react";
import { cn } from "@/lib/utils";
import { Separator } from "@/components/ui/separator";
import { ConnectionPicker } from "@/components/connection/connection-picker";
import { ConnectionDialog } from "@/components/connection/connection-dialog";

// The menu, in one place: route, label, icon.
const NAV = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "/query", label: "Query", icon: Database, end: false },
  { to: "/dml", label: "DML", icon: PencilRuler, end: false },
  { to: "/ddl", label: "DDL", icon: Activity, end: false },
  { to: "/analytics", label: "Analytics", icon: BarChart3, end: false },
];

export function AppShell() {
  return (
    <div className="flex min-h-svh bg-background text-foreground">
      <aside className="flex w-56 shrink-0 flex-col gap-1 border-r border-border p-3">
        <div className="px-2 py-3 text-sm font-semibold tracking-tight">noxis</div>
        <nav className="flex flex-col gap-1">
          {NAV.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground",
                  isActive && "bg-accent font-medium text-accent-foreground",
                )
              }
            >
              <Icon className="size-4" />
              {label}
            </NavLink>
          ))}
        </nav>
        <Separator className="my-2" />
        <p className="px-2 text-xs text-muted-foreground">
          SQL console for databases you already run.
        </p>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center gap-3 border-b border-border px-4 py-3">
          <ConnectionPicker />
          <span className="flex-1" />
          <ConnectionDialog trigger="New connection" />
        </header>
        <main className="min-w-0 flex-1 p-4">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
