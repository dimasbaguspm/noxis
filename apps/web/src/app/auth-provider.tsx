import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, type Identity, type Level } from "@/lib/api";
import { accessToken, initAuth, loadAuthConfig, signout as doSignout } from "@/lib/auth";
import { AppShellSkeleton } from "@/components/layout/app-shell-skeleton";

// AuthProvider owns the auth bootstrap (local account and/or OIDC) and the
// console's permission view. The server is the enforcement point either way;
// this only decides what the console shows.

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

// AccountTarget is the part of an account the console's guards read.
interface AccountTarget {
  subject: string;
  role?: string;
}

const LEVEL_RANK: Record<string, number> = { read: 1, write: 2, admin: 3 };

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [authOn, setAuthOn] = useState(false);
  const [localOn, setLocalOn] = useState(false);
  const [oidcOn, setOidcOn] = useState(false);
  const [authenticated, setAuthenticated] = useState(false);
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [ready, setReady] = useState(false);

  const isCallback = window.location.pathname === "/auth/callback";
  useEffect(() => {
    (async () => {
      const config = await loadAuthConfig();
      initAuth(config);
      setAuthOn(Boolean(config.local || config.oidc));
      setLocalOn(Boolean(config.local));
      setOidcOn(Boolean(config.oidc));
      // the callback route owns its own exchange; bootstrapping here would fire
      // unauthenticated /external/v1/* fetches and bounce back mid-exchange
      if (isCallback) return;
      const token = await accessToken();
      if (token) {
        try {
          setIdentity(await api.me());
          setAuthenticated(true);
        } catch {
          /* request() sends a 401 to the login page */
        }
      }
      setReady(true);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const role = identity?.role;
  const isSuper = !authOn || role === "super";
  const isAdmin = !authOn || role === "super" || role === "admin";

  const canAccess = useCallback(
    (connection: string | undefined, need: Level) => {
      if (!authOn) return true;
      if (!identity) return false;
      if (identity.role === "super") return true;
      const have = (connection ? identity.connections?.[connection]?.level : "") ?? "";
      return (LEVEL_RANK[have] ?? 0) >= LEVEL_RANK[need];
    },
    [authOn, identity],
  );

  // canManage is the shared guard for the account controls: nobody edits
  // themselves and the env root is untouchable. The server enforces the same.
  const canManage = useCallback(
    (target: AccountTarget) => Boolean(identity) && target.subject !== identity?.sub && target.role !== "super",
    [identity],
  );

  // canBan: an admin acts on members only; the super admin may also act on admins.
  const canBan = useCallback(
    (target: AccountTarget) => canManage(target) && (isSuper || target.role === "member"),
    [canManage, isSuper],
  );

  // canSetRole: only the super admin promotes or demotes.
  const canSetRole = useCallback(
    (target: AccountTarget) => isSuper && canManage(target),
    [canManage, isSuper],
  );

  const signout = useCallback(async () => {
    await doSignout();
  }, []);

  const value = useMemo(
    () => ({
      authEnabled: authOn,
      localEnabled: localOn,
      oidcEnabled: oidcOn,
      authenticated,
      isSuper,
      isAdmin,
      identity,
      ready,
      canAccess,
      isInstanceAdmin: (connection: string) => canAccess(connection, "admin"),
      canBan,
      canSetRole,
      signout,
    }),
    [authOn, localOn, oidcOn, authenticated, identity, isSuper, isAdmin, ready, canAccess, canBan, canSetRole, signout],
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
