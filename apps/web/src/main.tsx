import { StrictMode, Suspense, lazy } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { AuthProvider } from "@/app/auth-provider";
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
const CreateInstancePage = lazy(() => import("@/pages/console/instances/create"));
const InstanceDetailTab = lazy(() => import("@/pages/console/instances/detail"));
const InstanceTablesTab = lazy(() => import("@/pages/console/instances/tables"));
const InstanceErdTab = lazy(() => import("@/pages/console/instances/erd"));
const InstanceDocumentsTab = lazy(() => import("@/pages/console/instances/documents"));
const InstanceKeysTab = lazy(() => import("@/pages/console/instances/keys"));
const InstanceAccessTab = lazy(() => import("@/pages/console/instances/access"));
const InstanceChangeQueue = lazy(() =>
  import("@/pages/console/instances/queues").then((m) => ({ default: m.InstanceChangeQueue })),
);
const InstanceRequestsQueue = lazy(() =>
  import("@/pages/console/instances/queues").then((m) => ({ default: m.InstanceRequestsQueue })),
);
const QueryPage = lazy(() => import("@/pages/console/query"));
const SqlErdPage = lazy(() => import("@/pages/console/erd"));
const DmlPage = lazy(() => import("@/pages/console/changes/dml"));
const DdlPage = lazy(() => import("@/pages/console/changes/ddl"));
const KeysPage = lazy(() => import("@/pages/console/keys"));
const SearchPage = lazy(() => import("@/pages/console/search"));
const ExplorePage = lazy(() => import("@/pages/console/explore"));
const RequestsPage = lazy(() => import("@/pages/console/requests"));
const RequestNewPage = lazy(() => import("@/pages/console/requests/new"));
const RequestDetailPage = lazy(() => import("@/pages/console/requests/detail"));
const ChangeDetail = lazy(() =>
  import("@/pages/console/changes/detail").then((m) => ({ default: m.ChangeDetailPage })),
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

// Every page path comes from the registry. Engine pages sit under /drivers/:driver, so the URL names the engine a page
// acts on; instances are addressed by id, and the account and admin sections keep their own paths.
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
      { path: routePath(PAGE_ROUTES.INSTANCES), element: <DashboardPage /> },
      { path: routePath(PAGE_ROUTES.INSTANCE_CREATE), element: <CreateInstancePage /> },
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
          { path: "changes/dml", element: <InstanceChangeQueue kind="dml" /> },
          { path: "changes/ddl", element: <InstanceChangeQueue kind="ddl" /> },
          { path: "requests", element: <InstanceRequestsQueue /> },
        ],
      },
      // the driver section: one page per capability the driver's module serves
      { path: routePath(patternOf("driver.query")), element: <QueryPage /> },
      { path: routePath(patternOf("driver.erd")), element: <SqlErdPage /> },
      { path: routePath(patternOf("driver.changes.dml")), element: <DmlPage /> },
      {
        path: routePath(patternOf("driver.changes.dml.detail")),
        element: <ChangeDetail kind="dml" />,
      },
      { path: routePath(patternOf("driver.changes.ddl")), element: <DdlPage /> },
      {
        path: routePath(patternOf("driver.changes.ddl.detail")),
        element: <ChangeDetail kind="ddl" />,
      },
      { path: routePath(patternOf("driver.keys")), element: <KeysPage /> },
      { path: routePath(patternOf("driver.documents")), element: <ExplorePage /> },
      { path: routePath(patternOf("driver.search")), element: <SearchPage /> },
      { path: routePath(patternOf("driver.requests")), element: <RequestsPage /> },
      { path: routePath(patternOf("driver.requests.new")), element: <RequestNewPage /> },
      { path: routePath(patternOf("driver.requests.detail")), element: <RequestDetailPage /> },
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
      // the retired shorthand paths and any unknown path, including the old /console section, land on the instance list
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
        <TooltipProvider>
          <Suspense fallback={<div className="p-6 text-sm text-muted-foreground">Loading…</div>}>
            <RouterProvider router={router} />
          </Suspense>
        </TooltipProvider>
        <Toaster position="bottom-right" />
      </AuthProvider>
    </QueryClientProvider>
  </StrictMode>,
);
