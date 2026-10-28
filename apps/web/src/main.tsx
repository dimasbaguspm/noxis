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
import { PAGE_ROUTES, patternOf, routePath } from "@/constants/routes";
import "./app.css";

// route-level splitting: monaco (query/changes) loads with the page that needs it
const DashboardPage = lazy(() => import("@/pages/console/instances"));
const InstanceLayout = lazy(() => import("@/pages/console/instances/layout"));
const InstanceDetailTab = lazy(() => import("@/pages/console/instances/detail"));
const InstanceTablesTab = lazy(() => import("@/pages/console/instances/tables"));
const InstanceErdTab = lazy(() => import("@/pages/console/instances/erd"));
const InstanceDocumentsTab = lazy(() => import("@/pages/console/instances/documents"));
const InstanceKeysTab = lazy(() => import("@/pages/console/instances/keys"));
const InstanceAccessTab = lazy(() => import("@/pages/console/instances/access"));
const QueryPage = lazy(() => import("@/pages/console/sql/query"));
const SqlErdPage = lazy(() => import("@/pages/console/sql/erd"));
const DmlPage = lazy(() => import("@/pages/console/sql/dml"));
const DdlPage = lazy(() => import("@/pages/console/sql/ddl"));
const MemcachePage = lazy(() => import("@/pages/console/memcache"));
const MongoExplorePage = lazy(() => import("@/pages/console/mongodb/explore"));
const MongoRequestsPage = lazy(() => import("@/pages/console/mongodb/requests"));
const MongoRequestNewPage = lazy(() => import("@/pages/console/mongodb/requests/new"));
const MongoRequestDetailPage = lazy(() => import("@/pages/console/mongodb/requests/detail"));
const ChangeDetail = lazy(() =>
  import("@/pages/console/sql/changes/detail").then((m) => ({ default: m.ChangeDetailPage })),
);
const AuthCallbackPage = lazy(() => import("@/pages/auth/callback"));
const LoginPage = lazy(() => import("@/pages/auth/login"));
const MyLayout = lazy(() => import("@/pages/my/layout"));
const MyProfilePage = lazy(() => import("@/pages/my/profile"));
const MySettingsPage = lazy(() => import("@/pages/my/settings"));
const MyAccessPage = lazy(() => import("@/pages/my/access"));
const AccessPage = lazy(() => import("@/pages/admin/access"));
const UsersPage = lazy(() => import("@/pages/admin/users"));
const UserDetailPage = lazy(() => import("@/pages/admin/users/detail"));

// Every page path comes from PAGE_ROUTES; the engine pages sit under the console section, account and admin at the root.
const router = createBrowserRouter([
  // OIDC callback and login stay outside the shell and auth guard.
  { path: routePath(PAGE_ROUTES.AUTH_CALLBACK), element: <AuthCallbackPage /> },
  { path: routePath(PAGE_ROUTES.LOGIN), element: <LoginPage /> },
  {
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      // the console section: engine pages and the instance list, all under one prefix
      {
        path: routePath(PAGE_ROUTES.CONSOLE),
        element: <Navigate to={PAGE_ROUTES.INSTANCES} replace />,
      },
      { path: routePath(PAGE_ROUTES.INSTANCES), element: <DashboardPage /> },
      {
        path: routePath(patternOf("console.instances.detail")),
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
      { path: routePath(PAGE_ROUTES.SQL.QUERY), element: <QueryPage /> },
      { path: routePath(PAGE_ROUTES.SQL.ERD), element: <SqlErdPage /> },
      { path: routePath(PAGE_ROUTES.SQL.DML), element: <DmlPage /> },
      {
        path: routePath(patternOf("console.sql.dml.detail")),
        element: <ChangeDetail kind="dml" />,
      },
      { path: routePath(PAGE_ROUTES.SQL.DDL), element: <DdlPage /> },
      {
        path: routePath(patternOf("console.sql.ddl.detail")),
        element: <ChangeDetail kind="ddl" />,
      },
      { path: routePath(PAGE_ROUTES.MEMCACHE.KEYS), element: <MemcachePage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.EXPLORE), element: <MongoExplorePage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.REQUESTS), element: <MongoRequestsPage /> },
      { path: routePath(PAGE_ROUTES.MONGODB.REQUEST_NEW), element: <MongoRequestNewPage /> },
      {
        path: routePath(patternOf("console.mongodb.requests.detail")),
        element: <MongoRequestDetailPage />,
      },
      // the account section keeps its own paths
      {
        path: routePath(PAGE_ROUTES.MY),
        element: <MyLayout />,
        children: [
          { index: true, element: <Navigate to={PAGE_ROUTES.MY_PROFILE} replace /> },
          { path: "profile", element: <MyProfilePage /> },
          { path: "settings", element: <MySettingsPage /> },
          { path: "access", element: <MyAccessPage /> },
        ],
      },
      { path: routePath(PAGE_ROUTES.ACCESS), element: <AccessPage /> },
      { path: routePath(PAGE_ROUTES.USERS), element: <UsersPage /> },
      { path: routePath(patternOf("admin.users.detail")), element: <UserDetailPage /> },
      // the retired shorthand paths, the origin root, and any unknown path all land on the console list
      { path: "profile", element: <Navigate to={PAGE_ROUTES.MY} replace /> },
      { path: "settings", element: <Navigate to={PAGE_ROUTES.MY_SETTINGS} replace /> },
      { path: "*", element: <Navigate to={PAGE_ROUTES.INSTANCES} replace /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <ActiveProvider>
          <TooltipProvider>
            <Suspense fallback={<div className="p-6 text-sm text-muted-foreground">Loading…</div>}>
              <RouterProvider router={router} />
            </Suspense>
          </TooltipProvider>
          <Toaster position="bottom-right" />
        </ActiveProvider>
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>,
);
