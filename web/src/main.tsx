import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
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

const router = createBrowserRouter([
  {
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "query", element: <QueryPage /> },
      { path: "dml", element: <DmlPage /> },
      { path: "dml/:id", element: <ChangeDetail kind="dml" /> },
      { path: "ddl", element: <DdlPage /> },
      { path: "ddl/:id", element: <ChangeDetail kind="ddl" /> },
      { path: "analytics", element: <AnalyticsPage /> },
      { path: "memcache", element: <MemcachePage /> },
      { path: "memcache/analytics", element: <MemcacheAnalyticsPage /> },
      { path: "mongodb", element: <MongoExplorePage /> },
      { path: "mongodb/requests", element: <MongoRequestsPage /> },
      { path: "mongodb/requests/new", element: <MongoRequestNewPage /> },
      { path: "mongodb/requests/:id", element: <MongoRequestDetailPage /> },
      { path: "mongodb/analytics", element: <MongoAnalyticsPage /> },
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);

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
