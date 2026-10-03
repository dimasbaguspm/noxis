import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, Navigate, RouterProvider } from "react-router-dom";
import { AppProvider } from "@/app/app-provider";
import { AppShell } from "@/components/layout/app-shell";
import { Toaster } from "@/components/ui/sonner";
import AnalyticsPage from "@/pages/analytics";
import DashboardPage from "@/pages/dashboard";
import DdlPage from "@/pages/ddl";
import DmlPage from "@/pages/dml";
import QueryPage from "@/pages/query";
import "./app.css";

const router = createBrowserRouter([
  {
    element: <AppShell />,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: "query", element: <QueryPage /> },
      { path: "dml", element: <DmlPage /> },
      { path: "ddl", element: <DdlPage /> },
      { path: "analytics", element: <AnalyticsPage /> },
      { path: "*", element: <Navigate to="/" replace /> },
    ],
  },
]);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AppProvider>
      <RouterProvider router={router} />
      <Toaster position="bottom-right" />
    </AppProvider>
  </StrictMode>,
);
