import { createContext, useContext, useMemo, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { api, type Identity, type Level } from "@/lib/api";
import {
  accessToken,
  initAuth,
  loadAuthConfig,
  signout as doSignout,
  type AuthConfig,
} from "@/lib/auth";
import { queryKeys } from "@/lib/query-client";
import { canAccess, canBan, canSetRole, type AccountTarget } from "@/lib/policy";
import { PAGE_ROUTES } from "@/constants/routes";
import { AppShellSkeleton } from "@/components/layout/app-shell-skeleton";

// AuthProvider owns the auth bootstrap (local account and/or OIDC) and the console's permission view.

interface AuthState {
  authEnabled: boolean;
  localEnabled: boolean;
  oidcEnabled: boolean;
  authenticated: boolean;
  isSuper: boolean;
  isAdmin: boolean;
  identity: Identity | null;
  ready: boolean;
  canAccess: (connection: string | undefined, need: Level) => boolean;
  isInstanceAdmin: (connection: string) => boolean;
  canBan: (target: AccountTarget) => boolean;
  canSetRole: (target: AccountTarget) => boolean;
  signout: () => Promise<void>;
}

// Session is the one fact the provider holds: the auth config the server advertises and who the caller is.
interface Session {
  config: AuthConfig;
  identity: Identity | null;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  // the callback route owns its own exchange, so it must not bootstrap a session mid-flight
  const isCallback = window.location.pathname === PAGE_ROUTES.AUTH_CALLBACK;

  const session = useQuery({
    queryKey: queryKeys.auth.session,
    queryFn: async (): Promise<Session> => {
      const config = await loadAuthConfig();
      initAuth(config);
      if (isCallback) return { config, identity: null };
      const token = await accessToken();
      if (!token) return { config, identity: null };
      try {
        return { config, identity: await api.me() };
      } catch {
        // request() already sends a rejected session to the login page
        return { config, identity: null };
      }
    },
    staleTime: Infinity,
    retry: false,
  });

  const config = session.data?.config;
  const identity = session.data?.identity ?? null;
  const authEnabled = Boolean(config?.local || config?.oidc);
  const ready = session.isSuccess || session.isError;
  const isSuper = !authEnabled || identity?.role === "super";
  const isAdmin = !authEnabled || identity?.role === "super" || identity?.role === "admin";

  const value = useMemo<AuthState>(
    () => ({
      authEnabled,
      localEnabled: Boolean(config?.local),
      oidcEnabled: Boolean(config?.oidc),
      authenticated: identity !== null,
      isSuper,
      isAdmin,
      identity,
      ready,
      canAccess: (connection, need) => canAccess(identity, authEnabled, connection, need),
      isInstanceAdmin: (connection) => canAccess(identity, authEnabled, connection, "admin"),
      canBan: (target) => canBan(identity, isSuper, target),
      canSetRole: (target) => canSetRole(identity, isSuper, target),
      signout: () => doSignout(),
    }),
    [authEnabled, config, identity, isSuper, isAdmin, ready],
  );

  if (!ready && !isCallback) {
    return <AppShellSkeleton />;
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
