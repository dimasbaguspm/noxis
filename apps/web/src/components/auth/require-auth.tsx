import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "@/app/auth-provider";

// RequireAuth routes unauthenticated callers to login; it does nothing in open mode.
export function RequireAuth({ children }: { children: ReactNode }) {
  const { authEnabled, authenticated } = useAuth();
  if (authEnabled && !authenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}
