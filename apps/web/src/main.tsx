import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "@/app/auth-provider";
import { ActiveProvider } from "@/hooks/use-connections";
import { queryClient } from "@/lib/query-client";
import { AppShell } from "@/components/layout/app-shell";
import { RequireAuth } from "@/components/auth/require-auth";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { PAGE_ROUTES, routePath } from "@/lib/routes";
import "./app.css";

// route-level splitting: monaco (query/changes) loads with the page that needs it
const DashboardPage = lazy(() => import("@/pages/dashboard"));
const InstanceLayout = lazy(() => import("@/pages/instance/layout"));
const InstanceDetailTab = lazy(() => import("@/pages/instance/detail"));
const InstanceTablesTab = lazy(() => import("@/pages/instance/tables"));
const InstanceErdTab = lazy(() => import("@/pages/instance/erd"));
const InstanceDocumentsTab = lazy(() => import("@/pages/instance/documents"));
const InstanceKeysTab = lazy(() => import("@/pages/instance/keys"));
const InstanceAccessTab = lazy(() => import("@/pages/instance/access"));
const InstanceDashboardPage = lazy(() => import("@/pages/instance-dashboard"));
const QueryPage = lazy(() => import("@/pages/query"));
const SqlErdPage = lazy(() => import("@/pages/sql-erd"));
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
const LoginPage = lazy(() => import("@/pages/login"));
const MyLayout = lazy(() => import("@/pages/my"));
const MyProfilePage = lazy(() => import("@/pages/my-profile"));
const MySettingsPage = lazy(() => import("@/pages/my-settings"));
const MyAccessPage = lazy(() => import("@/pages/my-access"));
const AccessPage = lazy(() => import("@/pages/access"));
const UsersPage = lazy(() => import("@/pages/users"));
const UserDetailPage = lazy(() => import("@/pages/user-detail"));

// Pages use family-prefixed routes except the dashboard; PAGE_ROUTES supplies their paths.
const router = createBrowserRouter([
  // OIDC callback and login stay outside the shell and auth guard.
    { path: "auth/callback", element: <AuthCallbackPage /> },
  { path: "login", element: <LoginPage /> },
  {
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <DashboardPage /> },
      {
        path: routePath(PAGE_ROUTES.MY),
        element: <MyLayout />,
        children: [
          { index: true, element: <MyProfilePage /> },
          { path: "settings", element: <MySettingsPage /> },
          { path: "access", element: <MyAccessPage /> },
        ],
      },
      { path: "profile", element: <Navigate to={PAGE_ROUTES.MY} replace /> },
      { path: "settings", element: <Navigate to={PAGE_ROUTES.MY_SETTINGS} replace /> },
      { path: routePath(PAGE_ROUTES.ACCESS), element: <AccessPage /> },
      { path: routePath(PAGE_ROUTES.USERS), element: <UsersPage /> },
      { path: `${routePath(PAGE_ROUTES.USERS)}/:subject`, element: <UserDetailPage /> },
      {
        path: routePath(PAGE_ROUTES.INSTANCE_DETAIL(":id")),
        element: <InstanceLayout />,
        children: [
          { index: true, element: <InstanceDetailTab /> },
          { path: "tables", element: <InstanceTablesTab /> },
          { path: "erd", element: <InstanceErdTab /> },
          { path: "documents", element: <InstanceDocumentsTab /> },
          { path: "keys", element: <InstanceKeysTab /> },
          { path: "access", element: <InstanceAccessTab /> },
        ],
      },
      { path: routePath(PAGE_ROUTES.INSTANCE_DASHBOARD(":id")), element: <InstanceDashboardPage /> },
      { path: routePath(PAGE_ROUTES.SQL.QUERY), element: <QueryPage /> },
      { path: routePath(PAGE_ROUTES.SQL.ERD), element: <SqlErdPage /> },
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
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <ActiveProvider>
          <TooltipProvider>
        <Suspense fallback={<div className="p-6 text-sm text-muted-foreground">loading…</div>}>
          <RouterProvider router={router} />
        </Suspense>
      </TooltipProvider>
          <Toaster position="bottom-right" />
        </ActiveProvider>
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>,
);
