import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, type Identity, type Level } from "@/lib/api";
import { currentUser, initAuth, loadAuthConfig, signin, signout as oidcSignout } from "@/lib/auth";

// AuthProvider owns the OIDC bootstrap and the console's permission view: in
// open mode authEnabled is false and every permission is granted; the server
// stays the enforcement point either way.

interface AuthState {
  authEnabled: boolean;
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
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [ready, setReady] = useState(false);

  const isCallback = window.location.pathname === "/auth/callback";
  useEffect(() => {
    (async () => {
      const config = await loadAuthConfig();
      initAuth(config);
      setAuthOn(config.enabled);
      if (!config.enabled) {
        setReady(true);
        return;
      }
      // the callback route owns its own auth; bootstrapping here would fire
      // unauthenticated /v1/* fetches and bounce back to sign-in mid-exchange
      if (isCallback) return;
      const user = await currentUser();
      if (!user || user.expired) {
        await signin();
        return;
      }
      try {
        setIdentity(await api.me());
      } catch {
        /* request() sends a 401 back to sign-in */
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
    await oidcSignout();
  }, []);

  const value = useMemo(
    () => ({ authEnabled: authOn, isAdmin, identity, ready, can, signout }),
    [authOn, identity, isAdmin, ready, can, signout],
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