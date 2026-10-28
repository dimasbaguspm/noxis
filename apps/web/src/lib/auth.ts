import { UserManager, WebStorageStateStore, type User } from "oidc-client-ts";
import { PAGE_ROUTES } from "@/constants/routes";

// Local accounts and OIDC Authorization Code + PKCE both produce bearer tokens.

export interface AuthConfig {
  local?: boolean;
  oidc?: { issuer: string; clientId: string };
}

// AuthSession owns the auth state: the advertised methods, the OIDC manager, and the local token.
export class AuthSession {
  private static readonly TOKEN_KEY = "noxis.token";
  private manager: UserManager | null = null;
  private localEnabled = false;
  private authRequired = false;

  private localToken(): string | null {
    return window.localStorage.getItem(AuthSession.TOKEN_KEY);
  }

  private setLocalToken(token: string | null): void {
    if (token) window.localStorage.setItem(AuthSession.TOKEN_KEY, token);
    else window.localStorage.removeItem(AuthSession.TOKEN_KEY);
  }

  async loadConfig(): Promise<AuthConfig> {
    try {
      const res = await fetch("/external/v1/auth/config");
      if (!res.ok) return { local: false };
      return (await res.json()) as AuthConfig;
    } catch {
      return { local: false };
    }
  }

  // init records the available methods and builds the UserManager once.
  init(config: AuthConfig): void {
    this.localEnabled = Boolean(config.local);
    this.authRequired = this.localEnabled || Boolean(config.oidc);
    if (this.manager || !config.oidc) return;
    this.manager = new UserManager({
      authority: config.oidc.issuer,
      client_id: config.oidc.clientId,
      redirect_uri: `${window.location.origin}${PAGE_ROUTES.AUTH_CALLBACK}`,
      post_logout_redirect_uri: window.location.origin,
      response_type: "code",
      scope: "openid profile email groups",
      automaticSilentRenew: true,
      userStore: new WebStorageStateStore({ store: window.localStorage }),
    });
  }

  // enabled reports whether a login is required at all (always true today; the server has no open mode).
  get enabled(): boolean {
    return this.authRequired;
  }

  get localAvailable(): boolean {
    return this.localEnabled;
  }

  get oidcAvailable(): boolean {
    return this.manager !== null;
  }

  async currentUser(): Promise<User | null> {
    return this.manager ? this.manager.getUser() : null;
  }

  // accessToken returns the local session token when present, else a live OIDC token (silently refreshing through the manager).
  async accessToken(): Promise<string | null> {
    const local = this.localToken();
    if (local) return local;
    if (!this.manager) return null;
    const user = await this.manager.getUser();
    if (!user || user.expired) return null;
    return user.access_token;
  }

  // clearSession drops stale credentials to stop repeated 401 redirects between login and /me.
  async clearSession(): Promise<void> {
    this.setLocalToken(null);
    if (this.manager) await this.manager.removeUser();
  }

  // localLogin exchanges the account's email + password for a session token.
  async localLogin(email: string, password: string): Promise<void> {
    const res = await fetch("/external/v1/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    const data = (await res.json().catch(() => null)) as { token?: string; error?: string } | null;
    if (!res.ok || !data?.token) throw new Error(data?.error ?? `HTTP ${res.status}`);
    this.setLocalToken(data.token);
  }

  // changePassword rotates the password and invalidates the active session.
  async changePassword(oldPassword: string, newPassword: string): Promise<void> {
    const token = this.localToken();
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
    this.setLocalToken(null); // the server rotated the token; force a fresh login
  }

  async signin(): Promise<void> {
    if (this.manager) await this.manager.signinRedirect();
  }

  async handleCallback(): Promise<void> {
    if (!this.manager) throw new Error("OIDC is not configured");
    await this.manager.signinRedirectCallback();
  }

  // signout clears the local session and, when OIDC is configured, ends that too.
  async signout(): Promise<void> {
    const token = this.localToken();
    if (token) {
      await fetch("/external/v1/auth/logout", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => undefined);
      this.setLocalToken(null);
    }
    if (this.manager) {
      await this.manager.signoutRedirect();
      return;
    }
    window.location.assign(PAGE_ROUTES.LOGIN);
  }
}

// auth is the console's one session; callers use the wrappers below or the instance directly.
export const auth = new AuthSession();

export const loadAuthConfig = () => auth.loadConfig();
export const initAuth = (config: AuthConfig) => auth.init(config);
export const authEnabled = () => auth.enabled;
export const localAuthAvailable = () => auth.localAvailable;
export const oidcAuthAvailable = () => auth.oidcAvailable;
export const currentUser = () => auth.currentUser();
export const accessToken = () => auth.accessToken();
export const clearSession = () => auth.clearSession();
export const localLogin = (email: string, password: string) => auth.localLogin(email, password);
export const changePassword = (oldPassword: string, newPassword: string) =>
  auth.changePassword(oldPassword, newPassword);
export const signin = () => auth.signin();
export const handleCallback = () => auth.handleCallback();
export const signout = () => auth.signout();
