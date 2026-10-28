import { Outlet, Link, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { CrumbLabelProvider } from "@/app/crumb-provider";
import { NavSearch } from "@/components/layout/nav-search";
import { AppSidebar } from "@/components/layout/app-sidebar";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { useBreadcrumbs } from "@/hooks/use-breadcrumbs";
import { useDocumentTitle } from "@/hooks/use-document-title";

// AppShell supplies the crumb registry, so a page can name its own record while the header renders the trail.
export function AppShell() {
  return (
    <CrumbLabelProvider>
      <Shell />
    </CrumbLabelProvider>
  );
}

// Shell is the layout itself: sidebar, header with the path trail and search, and the route outlet.
function Shell() {
  const { pathname } = useLocation();
  const crumbs = useBreadcrumbs();
  useDocumentTitle();

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset className="h-svh overflow-hidden">
        <header className="sticky top-0 z-10 flex items-center gap-2 border-b border-border bg-background/95 px-4 py-2 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <SidebarTrigger className="-ms-2" />
          <nav aria-label="Breadcrumb" className="min-w-0 flex-1 overflow-hidden">
            <ol className="flex min-w-0 items-center gap-1.5 text-sm">
              {crumbs.map((crumb, i) => (
                <li
                  key={`${crumb.to ?? "leaf"}-${i}`}
                  className={cn(
                    "flex min-w-0 items-center gap-1.5 whitespace-nowrap",
                    i < crumbs.length - 1 && "hidden sm:flex",
                  )}
                >
                  {i > 0 && <span className="text-muted-foreground/50">/</span>}
                  {crumb.to && crumb.to !== pathname ? (
                    <Link to={crumb.to} className="truncate text-muted-foreground hover:underline">
                      {crumb.label}
                    </Link>
                  ) : (
                    <span
                      className={cn(
                        "truncate",
                        i === crumbs.length - 1 ? "font-medium" : "text-muted-foreground",
                      )}
                    >
                      {crumb.label}
                    </span>
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
