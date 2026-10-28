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
      {
        path: `${routePath(PAGE_ROUTES.MONGODB.REQUESTS)}/:id`,
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
      { path: `${routePath(PAGE_ROUTES.USERS)}/:subject`, element: <UserDetailPage /> },
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
