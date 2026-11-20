import axios from 'axios';
import type { AxiosInstance, AxiosRequestConfig } from 'axios';

// One HTTP client for the console: the envelope, the bearer header, the token cookie, and an error a
// screen can render. axios rather than fetch because retries, cancellation and response transforms
// are the three things every screen needs and nobody wants to write twice.

const TOKEN_COOKIE = 'noxis_token';

export type Envelope<T> = {
  data?: T;
  meta?: { next_cursor?: string };
  error?: { code: string; message: string };
};

export type Page<T> = { items: T[]; next?: string };

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

export function readToken(): string {
  const match = document.cookie.match(new RegExp(`(?:^|; )${TOKEN_COOKIE}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : '';
}

export function writeToken(token: string): void {
  const secure = location.protocol === 'https:' ? '; Secure' : '';
  document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
}

export function clearToken(): void {
  document.cookie = `${TOKEN_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
}

// config.json is served by the binary that carried this build: it knows the API the browser reaches,
// and — when it was started from a signed-in CLI — the token that session already holds.
export type Config = { api: string; token?: string };

let configPromise: Promise<Config> | null = null;

export function loadConfig(): Promise<Config> {
  const fallback: Config = { api: import.meta.env.VITE_API_URL ?? 'http://localhost:9000' };
  // An absolute path, not `./config.json`: the console owns deep links of its own, and on `/targets/:ref`
  // a relative fetch asks the server for `/targets/config.json`, which the SPA fallback answers with the
  // shell as HTML. The shape is checked too, so a wrong answer falls back instead of becoming a config
  // with no API in it.
  configPromise ??= axios
    .get<Config>('/config.json', { headers: { Accept: 'application/json' } })
    .then((response) => (typeof response.data?.api === 'string' ? response.data : fallback))
    .catch(() => fallback);
  return configPromise;
}

// client is one axios instance per API base, so a screen passes a hook rather than a URL.
export function createClient(api: string, token?: string): AxiosInstance {
  const instance = axios.create({
    baseURL: api,
    timeout: 60_000,
    headers: { Accept: 'application/json' },
  });

  instance.interceptors.request.use((config) => {
    const bearer = token || readToken();
    if (bearer) config.headers.Authorization = `Bearer ${bearer}`;
    return config;
  });

  instance.interceptors.response.use(
    (response) => response,
    (error: unknown) => {
      if (!axios.isAxiosError(error)) return Promise.reject(error);
      const status = error.response?.status ?? 0;
      if (status === 401 && !token) clearToken();
      const envelope = error.response?.data as Envelope<unknown> | undefined;
      const message = envelope?.error?.message ?? error.message ?? 'the API did not answer';
      return Promise.reject(new ApiError(status, envelope?.error?.code ?? 'network', message));
    },
  );

  return instance;
}

// unwrap is the only place that knows the envelope: a hook returns `data`, never the wrapper.
export async function unwrap<T>(call: Promise<{ data: Envelope<T> }>): Promise<T> {
  const { data } = await call;
  return data.data as T;
}

export async function unwrapPage<T>(call: Promise<{ data: Envelope<T[]> }>): Promise<Page<T>> {
  const { data } = await call;
  return { items: data.data ?? [], next: data.meta?.next_cursor };
}

export type ListTargets = {
  limit?: number;
  cursor?: string;
  q?: string;
  driver?: string;
  include_archived?: boolean;
};

export type QueryList = Record<string, string | number | boolean | undefined>;

export function params(list: QueryList): AxiosRequestConfig {
  const search: Record<string, string> = {};
  for (const [key, value] of Object.entries(list)) {
    if (value === undefined || value === '' || value === false) continue;
    search[key] = String(value);
  }
  return { params: search };
}
