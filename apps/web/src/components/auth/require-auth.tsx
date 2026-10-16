import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";
import { useAuth } from "@/app/auth-provider";

// RequireAuth guards the shell: with auth enabled but no valid session the
// router shows the login page instead of any engine screen. Open mode never
// trips it (authEnabled is false).
export function RequireAuth({ children }: { children: ReactNode }) {
  const { authEnabled, authenticated } = useAuth();
  if (authEnabled && !authenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}