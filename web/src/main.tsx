import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider, useParams } from "react-router-dom";
import { AppProvider } from "@/app/app-provider";
import { AppShell } from "@/components/layout/app-shell";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
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
// dashboard is the only family-less route.
const router = createBrowserRouter([
  {
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "sql/query", element: <QueryPage /> },
      { path: "sql/dml", element: <DmlPage /> },
      { path: "sql/dml/:id", element: <ChangeDetail kind="dml" /> },
      { path: "sql/ddl", element: <DdlPage /> },
      { path: "sql/ddl/:id", element: <ChangeDetail kind="ddl" /> },
      { path: "sql/analytics", element: <AnalyticsPage /> },
      { path: "memcache", element: <MemcachePage /> },
      { path: "memcache/analytics", element: <MemcacheAnalyticsPage /> },
      { path: "mongodb", element: <MongoExplorePage /> },
      { path: "mongodb/requests", element: <MongoRequestsPage /> },
      { path: "mongodb/requests/new", element: <MongoRequestNewPage /> },
      { path: "mongodb/requests/:id", element: <MongoRequestDetailPage /> },
      { path: "mongodb/analytics", element: <MongoAnalyticsPage /> },
      // pre-family SQL URLs keep working for bookmarks and old links
      { path: "query", element: <Moved to="/sql/query" /> },
      { path: "dml", element: <Moved to="/sql/dml" /> },
      { path: "dml/:id", element: <Moved to="/sql/dml" keepId /> },
      { path: "ddl", element: <Moved to="/sql/ddl" /> },
      { path: "ddl/:id", element: <Moved to="/sql/ddl" keepId /> },
      { path: "analytics", element: <Moved to="/sql/analytics" /> },
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);

// Moved sends an old family-less SQL URL to its /sql/ equivalent, keeping the
// detail id so a bookmarked change still opens.
function Moved({ to, keepId = false }: { to: string; keepId?: boolean }) {
  const { id } = useParams();
  return <Navigate to={keepId && id ? `${to}/${id}` : to} replace />;
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
