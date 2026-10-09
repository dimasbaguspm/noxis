import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { AppProvider } from "@/app/app-provider";
import { AuthProvider } from "@/app/auth-provider";
import { AppShell } from "@/components/layout/app-shell";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { PAGE_ROUTES, routePath } from "@/lib/routes";
import "./app.css";

// route-level splitting: monaco (query/changes) loads with the page that needs it
const DashboardPage = lazy(() => import("@/pages/dashboard"));
const InstanceDetailPage = lazy(() => import("@/pages/instance-detail"));
const QueryPage = lazy(() => import("@/pages/query"));
const DmlPage = lazy(() => import("@/pages/dml"));
const DdlPage = lazy(() => import("@/pages/ddl"));
const MemcachePage = lazy(() => import("@/pages/memcache"));
const MongoExplorePage = lazy(() => import("@/pages/mongodb"));
const MongoRequestsPage = lazy(() => import("@/pages/mongodb-requests"));
const MongoRequestNewPage = lazy(() => import("@/pages/mongodb-request-new"));
const MongoRequestDetailPage = lazy(() => import("@/pages/mongodb-request-detail"));
const ChangeDetail = lazy(() =>
  import("@/pages/change-detail").then((m) => ({ default: m.ChangeDetailPage })),
);
const AuthCallbackPage = lazy(() => import("@/pages/auth-callback"));

// Every page carries its engine family prefix (/sql, /memcache, /mongodb); the
// dashboard is the only family-less route. Paths come from PAGE_ROUTES.
const router = createBrowserRouter([
  // the OIDC redirect target sits outside the shell: it has no chrome and
  // leaves the page as soon as the code is exchanged
  { path: "auth/callback", element: <AuthCallbackPage /> },
  {
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: routePath(PAGE_ROUTES.INSTANCE_DETAIL(":id")), element: <InstanceDetailPage /> },
      { path: routePath(PAGE_ROUTES.SQL.QUERY), element: <QueryPage /> },
      { path: routePath(PAGE_ROUTES.SQL.DML), element: <DmlPage /> },
      { path: `${routePath(PAGE_ROUTES.SQL.DML)}/:id`, element: <ChangeDetail kind="dml" /> },
      { path: routePath(PAGE_ROUTES.SQL.DDL), element: <DdlPage /> },
      { path: `${routePath(PAGE_ROUTES.SQL.DDL)}/:id`, element: <ChangeDetail kind="ddl" /> },
      { path: routePath(PAGE_ROUTES.MEMCACHE.KEYS), element: <MemcachePage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.EXPLORE), element: <MongoExplorePage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.REQUESTS), element: <MongoRequestsPage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.REQUEST_NEW), element: <MongoRequestNewPage /> },
      { path: `${routePath(PAGE_ROUTES.MONGODB.REQUESTS)}/:id`, element: <MongoRequestDetailPage /> },
      { path: "*", element: <Navigate to={PAGE_ROUTES.DASHBOARD} replace /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AuthProvider>
      <AppProvider>
        <TooltipProvider>
        <Suspense fallback={<div className="p-6 text-sm text-muted-foreground">loading…</div>}>
          <RouterProvider router={router} />
        </Suspense>
      </TooltipProvider>
        <Toaster position="bottom-right" />
      </AppProvider>
    </AuthProvider>
  </StrictMode>,
);
