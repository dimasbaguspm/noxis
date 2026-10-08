import { UserManager, WebStorageStateStore, type User } from "oidc-client-ts";

// The console is a public OIDC client (Authorization Code + PKCE, no secret).
// The server exposes the issuer and client id at /auth/config so the same
// image can point at a different Authentik without a rebuild.

export interface AuthConfig {
  enabled: boolean;
  issuer?: string;
  clientId?: string;
}

let manager: UserManager | null = null;

export async function loadAuthConfig(): Promise<AuthConfig> {
  try {
    const res = await fetch("/auth/config");
    if (!res.ok) return { enabled: false };
    return (await res.json()) as AuthConfig;
  } catch {
    return { enabled: false };
  }
}

// initAuth builds the UserManager once; open mode leaves it null.
export function initAuth(config: AuthConfig): void {
  if (manager || !config.enabled || !config.issuer || !config.clientId) return;
  manager = new UserManager({
    authority: config.issuer,
    client_id: config.clientId,
    redirect_uri: `${window.location.origin}/auth/callback`,
    post_logout_redirect_uri: window.location.origin,
    response_type: "code",
    scope: "openid profile email groups",
    automaticSilentRenew: true,
    userStore: new WebStorageStateStore({ store: window.localStorage }),
  });
}

export function authEnabled(): boolean {
  return manager !== null;
}

export async function currentUser(): Promise<User | null> {
  return manager ? manager.getUser() : null;
}

// accessToken returns a live token, silently refreshing through the manager.
export async function accessToken(): Promise<string | null> {
  if (!manager) return null;
  const user = await manager.getUser();
  if (!user || user.expired) return null;
  return user.access_token;
}

export async function signin(): Promise<void> {
  if (manager) await manager.signinRedirect();
}

export async function handleCallback(): Promise<void> {
  if (manager) await manager.signinRedirectCallback();
}

export async function signout(): Promise<void> {
  if (manager) await manager.signoutRedirect();
}
