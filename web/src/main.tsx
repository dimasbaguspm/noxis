import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { AppProvider } from "@/app/app-provider";
import { PLACEHOLDERS } from "@/lib/nav";
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
const RedisExplorePage = lazy(() => import("@/pages/redis-explore"));
const RedisAnalyticsPage = lazy(() => import("@/pages/redis-analytics"));
const PlaceholderPage = lazy(() =>
  import("@/components/layout/placeholder-page").then((m) => ({ default: m.PlaceholderPage })),
);
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
      { path: "redis", element: <RedisExplorePage /> },
      { path: "redis/analytics", element: <RedisAnalyticsPage /> },
      ...Object.keys(PLACEHOLDERS).map((path) => ({
        path: path.replace(/^\//, ""),
        element: <PlaceholderPage spec={PLACEHOLDERS[path]} />,
      })),
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
