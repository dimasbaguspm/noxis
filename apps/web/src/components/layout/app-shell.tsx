import { Outlet, Link } from "react-router-dom";
import { cn } from "@/lib/utils";
import { NavSearch } from "@/components/layout/nav-search";
import { AppSidebar } from "@/components/layout/app-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { useBreadcrumbs } from "@/hooks/use-breadcrumbs";

// AppShell is the sidebar + header frame every engine page renders inside:
// the shadcn sidebar structure on the left, an inset column with breadcrumbs,
// the Cmd+K page search and the routed outlet.
export function AppShell() {
  const crumbs = useBreadcrumbs();

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="h-svh overflow-hidden">
        <header className="sticky top-0 z-10 flex items-center gap-2 border-b border-border bg-background/95 px-4 py-2 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <SidebarTrigger className="-ms-2" />
          <nav aria-label="breadcrumb" className="min-w-0 flex-1 overflow-hidden">
            <ol className="flex min-w-0 items-center gap-1.5 text-sm">
              {crumbs.map((crumb, i) => (
                <li
                  key={`${crumb.label}-${i}`}
                  className={cn(
                    "flex min-w-0 items-center gap-1.5 whitespace-nowrap",
                    i < crumbs.length - 1 && "hidden sm:flex",
                  )}
                >
                  {i > 0 && <span className="text-muted-foreground/50">/</span>}
                  {crumb.to && i < crumbs.length - 1 ? (
                    <Link to={crumb.to} className="text-muted-foreground hover:underline">
                      {crumb.label}
                    </Link>
                  ) : (
                    <span className="truncate font-medium">{crumb.label}</span>
                  )}
                </li>
              ))}
            </ol>
          </nav>
          <NavSearch />
        </header>
        <main className="min-h-0 min-w-0 flex-1 overflow-y-auto p-4">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}