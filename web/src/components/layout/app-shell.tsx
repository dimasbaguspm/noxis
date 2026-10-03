import { NavLink, Outlet, Link } from "react-router-dom";
import { Activity, BarChart3, Database, LayoutDashboard, Leaf, PencilRuler } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useBreadcrumbs } from "@/hooks/use-breadcrumbs";
import { cn } from "@/lib/utils";

type NavItem = { to: string; label: string; icon: typeof LayoutDashboard; end?: boolean; soon?: boolean };
type NavGroup = { label?: string; items: NavItem[] };

// The menu: Dashboard and Analytics at the top, SQL work grouped, and the
// next engines parked as "soon" so the shape of the product is visible.
const NAV: NavGroup[] = [
  {
    items: [
      { to: "/", label: "Dashboard", icon: LayoutDashboard, end: true },
      { to: "/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
  {
    label: "SQL",
    items: [
      { to: "/query", label: "Query", icon: Database },
      { to: "/dml", label: "DML", icon: PencilRuler },
      { to: "/ddl", label: "DDL", icon: Activity },
    ],
  },
  {
    label: "MongoDB",
    items: [{ to: "/mongo", label: "Documents", icon: Leaf, soon: true }],
  },
];

export function AppShell() {
  const crumbs = useBreadcrumbs();

  return (
    <div className="flex min-h-svh bg-background text-foreground">
      <aside className="flex w-56 shrink-0 flex-col gap-3 border-r border-border p-3">
        <div className="px-2 py-2 text-sm font-semibold tracking-tight">noxis</div>
        {NAV.map((group, i) => (
          <div key={group.label ?? `group-${i}`} className="flex flex-col gap-1">
            {group.label && (
              <>
                {i > 0 && <Separator className="mb-1" />}
                <p className="px-2 text-[11px] font-medium tracking-wider text-muted-foreground uppercase">
                  {group.label}
                </p>
              </>
            )}
            {group.items.map(({ to, label, icon: Icon, end, soon }) =>
              soon ? (
                <span
                  key={to}
                  className="flex cursor-not-allowed items-center gap-2 rounded-md px-2.5 py-1.5 text-sm text-muted-foreground/60"
                >
                  <Icon className="size-4" />
                  {label}
                  <Badge variant="outline" className="ml-auto text-[10px]">
                    soon
                  </Badge>
                </span>
              ) : (
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
              ),
            )}
          </div>
        ))}
        <span className="flex-1" />
        <p className="px-2 text-xs text-muted-foreground">SQL console for databases you already run.</p>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center gap-2 border-b border-border px-4 py-2.5">
          <nav aria-label="breadcrumb">
            <ol className="flex items-center gap-1.5 text-sm">
              {crumbs.map((crumb, i) => (
                <li key={`${crumb.label}-${i}`} className="flex items-center gap-1.5">
                  {i > 0 && <span className="text-muted-foreground/50">/</span>}
                  {crumb.to && i < crumbs.length - 1 ? (
                    <Link to={crumb.to} className="text-muted-foreground hover:underline">
                      {crumb.label}
                    </Link>
                  ) : (
                    <span className="font-medium">{crumb.label}</span>
                  )}
                </li>
              ))}
            </ol>
          </nav>
        </header>
        <main className="min-w-0 flex-1 p-4">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
