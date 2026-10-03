import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { AppProvider } from "@/app/app-provider";
import { AppShell } from "@/components/layout/app-shell";
import { Toaster } from "@/components/ui/sonner";
import "./app.css";

// route-level splitting: monaco (query/changes) and recharts (analytics)
// load with the page that needs them
const DashboardPage = lazy(() => import("@/pages/dashboard"));
const QueryPage = lazy(() => import("@/pages/query"));
const DmlPage = lazy(() => import("@/pages/dml"));
const DdlPage = lazy(() => import("@/pages/ddl"));
const AnalyticsPage = lazy(() => import("@/pages/analytics"));
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
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppProvider>
      <Suspense fallback={<div className="p-6 text-sm text-muted-foreground">loading…</div>}>
        <RouterProvider router={router} />
      </Suspense>
      <Toaster position="bottom-right" />
    </AppProvider>
  </StrictMode>,
);
