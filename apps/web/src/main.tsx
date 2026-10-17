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
const LoginPage = lazy(() => import("@/pages/login"));
const SettingsPage = lazy(() => import("@/pages/settings"));
const ProfilePage = lazy(() => import("@/pages/profile"));
const UsersPage = lazy(() => import("@/pages/users"));
const UserDetailPage = lazy(() => import("@/pages/user-detail"));

// Every page carries its engine family prefix (/sql, /memcache, /mongodb); the
// dashboard is the only family-less route. Paths come from PAGE_ROUTES.
const router = createBrowserRouter([
  // the OIDC redirect target and the login page sit outside the shell: no
  // chrome, no auth guard on them
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
      { path: routePath(PAGE_ROUTES.SETTINGS), element: <SettingsPage /> },
      { path: routePath(PAGE_ROUTES.PROFILE), element: <ProfilePage /> },
      { path: routePath(PAGE_ROUTES.USERS), element: <UsersPage /> },
      { path: `${routePath(PAGE_ROUTES.USERS)}/:subject`, element: <UserDetailPage /> },
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
