import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, type Identity, type Level } from "@/lib/api";
import { accessToken, initAuth, loadAuthConfig, signout as doSignout } from "@/lib/auth";

// AuthProvider owns the auth bootstrap (local account and/or OIDC) and the
// console's permission view. The server is the enforcement point either way;
// this only decides what the console shows.

interface AuthState {
  authEnabled: boolean;
  localEnabled: boolean;
  oidcEnabled: boolean;
  authenticated: boolean;
  isAdmin: boolean;
  identity: Identity | null;
  ready: boolean;
  can: (connection: string | undefined, need: Level) => boolean;
  signout: () => Promise<void>;
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

  const isAdmin = !authOn || (identity?.admin ?? false);

  const can = useCallback(
    (connection: string | undefined, need: Level) => {
      if (!authOn) return true;
      if (!identity) return false;
      if (identity.admin) return true;
      const have = (connection ? identity.connections?.[connection] : "") ?? "";
      return (LEVEL_RANK[have] ?? 0) >= LEVEL_RANK[need];
    },
    [authOn, identity],
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
      isAdmin,
      identity,
      ready,
      can,
      signout,
    }),
    [authOn, localOn, oidcOn, authenticated, identity, isAdmin, ready, can, signout],
  );

  if (!ready && !isCallback) {
    return (
      <div className="flex h-svh items-center justify-center text-sm text-muted-foreground">
        signing in…
      </div>
    );
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
