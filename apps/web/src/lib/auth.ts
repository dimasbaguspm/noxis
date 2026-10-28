import { UserManager, WebStorageStateStore, type User } from "oidc-client-ts";

// Local accounts and OIDC Authorization Code + PKCE both produce bearer tokens.

export interface AuthConfig {
  local?: boolean;
  oidc?: { issuer: string; clientId: string };
}

const LOCAL_TOKEN_KEY = "noxis.token";

let manager: UserManager | null = null;
let localEnabled = false;
let authRequired = false;

function localToken(): string | null {
  return window.localStorage.getItem(LOCAL_TOKEN_KEY);
}

function setLocalToken(token: string | null): void {
  if (token) window.localStorage.setItem(LOCAL_TOKEN_KEY, token);
  else window.localStorage.removeItem(LOCAL_TOKEN_KEY);
}

export async function loadAuthConfig(): Promise<AuthConfig> {
  try {
    const res = await fetch("/external/v1/auth/config");
    if (!res.ok) return { local: false };
    return (await res.json()) as AuthConfig;
  } catch {
    return { local: false };
  }
}

// initAuth records the available methods and builds the UserManager once.
export function initAuth(config: AuthConfig): void {
  localEnabled = Boolean(config.local);
  authRequired = localEnabled || Boolean(config.oidc);
  if (manager || !config.oidc) return;
  manager = new UserManager({
    authority: config.oidc.issuer,
    client_id: config.oidc.clientId,
    redirect_uri: `${window.location.origin}/auth/callback`,
    post_logout_redirect_uri: window.location.origin,
    response_type: "code",
    scope: "openid profile email groups",
    automaticSilentRenew: true,
    userStore: new WebStorageStateStore({ store: window.localStorage }),
  });
}

// authEnabled reports whether a login is required at all (always true today; the server has no open mode).
export function authEnabled(): boolean {
  return authRequired;
}

export function localAuthAvailable(): boolean {
  return localEnabled;
}

export function oidcAuthAvailable(): boolean {
  return manager !== null;
}

export async function currentUser(): Promise<User | null> {
  return manager ? manager.getUser() : null;
}

// accessToken returns the local session token when present, else a live OIDC token (silently refreshing through the manager).
export async function accessToken(): Promise<string | null> {
  const local = localToken();
  if (local) return local;
  if (!manager) return null;
  const user = await manager.getUser();
  if (!user || user.expired) return null;
  return user.access_token;
}

// clearSession drops stale credentials to stop repeated 401 redirects between login and /me.
export async function clearSession(): Promise<void> {
  setLocalToken(null);
  if (manager) await manager.removeUser();
}

// localLogin exchanges the account's email + password for a session token.
export async function localLogin(email: string, password: string): Promise<void> {
  const res = await fetch("/external/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const data = (await res.json().catch(() => null)) as { token?: string; error?: string } | null;
  if (!res.ok || !data?.token) throw new Error(data?.error ?? `HTTP ${res.status}`);
  setLocalToken(data.token);
}

// changePassword rotates the password and invalidates the active session.
export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  const token = localToken();
  const res = await fetch("/external/v1/auth/password", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ oldPassword, newPassword }),
  });
  if (!res.ok) {
    const data = (await res.json().catch(() => null)) as { error?: string } | null;
    throw new Error(data?.error ?? `HTTP ${res.status}`);
  }
  setLocalToken(null); // the server rotated the token; force a fresh login
}

export async function signin(): Promise<void> {
  if (manager) await manager.signinRedirect();
}

export async function handleCallback(): Promise<void> {
  if (!manager) throw new Error("OIDC is not configured");
  await manager.signinRedirectCallback();
}

// signout clears the local session and, when OIDC is configured, ends that too.
export async function signout(): Promise<void> {
  const token = localToken();
  if (token) {
    await fetch("/external/v1/auth/logout", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
    }).catch(() => undefined);
    setLocalToken(null);
  }
  if (manager) {
    await manager.signoutRedirect();
    return;
  }
  window.location.assign("/login");
}
