import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider, useParams, useSearchParams } from "react-router-dom";
import { AppProvider } from "@/app/app-provider";
import { AppShell } from "@/components/layout/app-shell";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { LEGACY_SQL, PAGE_ROUTES, routePath } from "@/lib/routes";
import "./app.css";

// route-level splitting: monaco (query/changes) and recharts (analytics)
// load with the page that needs them
const DashboardPage = lazy(() => import("@/pages/dashboard"));
const QueryPage = lazy(() => import("@/pages/query"));
const DmlPage = lazy(() => import("@/pages/dml"));
const DdlPage = lazy(() => import("@/pages/ddl"));
const AnalyticsPage = lazy(() => import("@/pages/analytics"));
const MemcachePage = lazy(() => import("@/pages/memcache"));
const MemcacheAnalyticsPage = lazy(() => import("@/pages/memcache-analytics"));
const MongoExplorePage = lazy(() => import("@/pages/mongodb"));
const MongoRequestsPage = lazy(() => import("@/pages/mongodb-requests"));
const MongoRequestNewPage = lazy(() => import("@/pages/mongodb-request-new"));
const MongoRequestDetailPage = lazy(() => import("@/pages/mongodb-request-detail"));
const MongoAnalyticsPage = lazy(() => import("@/pages/mongodb-analytics"));
const ChangeDetail = lazy(() =>
  import("@/pages/change-detail").then((m) => ({ default: m.ChangeDetailPage })),
);

// Every page carries its engine family prefix (/sql, /memcache, /mongodb); the
// dashboard is the only family-less route. Paths come from PAGE_ROUTES.
const router = createBrowserRouter([
  {
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: routePath(PAGE_ROUTES.SQL.QUERY), element: <QueryPage /> },
      { path: routePath(PAGE_ROUTES.SQL.DML), element: <DmlPage /> },
      { path: `${routePath(PAGE_ROUTES.SQL.DML)}/:id`, element: <ChangeDetail kind="dml" /> },
      { path: routePath(PAGE_ROUTES.SQL.DDL), element: <DdlPage /> },
      { path: `${routePath(PAGE_ROUTES.SQL.DDL)}/:id`, element: <ChangeDetail kind="ddl" /> },
      { path: routePath(PAGE_ROUTES.SQL.ANALYTICS), element: <AnalyticsPage /> },
      { path: routePath(PAGE_ROUTES.MEMCACHE.KEYS), element: <MemcachePage /> },
      { path: routePath(PAGE_ROUTES.MEMCACHE.ANALYTICS), element: <MemcacheAnalyticsPage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.EXPLORE), element: <MongoExplorePage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.REQUESTS), element: <MongoRequestsPage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.REQUEST_NEW), element: <MongoRequestNewPage /> },
      { path: `${routePath(PAGE_ROUTES.MONGODB.REQUESTS)}/:id`, element: <MongoRequestDetailPage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.ANALYTICS), element: <MongoAnalyticsPage /> },
      // pre-family SQL URLs keep working for bookmarks and old links
      { path: LEGACY_SQL.QUERY, element: <Moved to={PAGE_ROUTES.SQL.QUERY} /> },
      { path: LEGACY_SQL.DML, element: <Moved to={PAGE_ROUTES.SQL.DML} /> },
      { path: LEGACY_SQL.DML_DETAIL, element: <Moved to={PAGE_ROUTES.SQL.DML} keepId /> },
      { path: LEGACY_SQL.DDL, element: <Moved to={PAGE_ROUTES.SQL.DDL} /> },
      { path: LEGACY_SQL.DDL_DETAIL, element: <Moved to={PAGE_ROUTES.SQL.DDL} keepId /> },
      { path: LEGACY_SQL.ANALYTICS, element: <Moved to={PAGE_ROUTES.SQL.ANALYTICS} /> },
      { path: "*", element: <Navigate to={PAGE_ROUTES.DASHBOARD} replace /> },
    ],
  },
]);

// Moved sends an old family-less SQL URL to its /sql/ equivalent, keeping the
// detail id and any querystring so a bookmarked change still opens.
function Moved({ to, keepId = false }: { to: string; keepId?: boolean }) {
  const { id } = useParams();
  const [search] = useSearchParams();
  const path = keepId && id ? `${to}/${id}` : to;
  const qs = search.toString();
  return <Navigate to={qs ? `${path}?${qs}` : path} replace />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppProvider>
      <TooltipProvider>
        <Suspense fallback={<div className="p-6 text-sm text-muted-foreground">loading…</div>}>
          <RouterProvider router={router} />
        </Suspense>
      </TooltipProvider>
      <Toaster position="bottom-right" />
    </AppProvider>
  </StrictMode>,
);
