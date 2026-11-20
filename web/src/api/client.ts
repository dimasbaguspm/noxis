import type {
  Capabilities,
  Identity,
  JournalEntry,
  Page,
  Proposal,
  ProposalDetail,
  Rows,
  Table,
  TableDetail,
  Target,
} from './types';

// The console's whole knowledge of the API: the envelope, the bearer header, the cookie the token
// lives in, and one function per route the console uses. Nothing here invents a route.

const TOKEN_COOKIE = 'noxis_token';

// api is always there. token is the CLI's own session, handed over only by a binary that was started
// on this machine by someone who already signed in — see `noxis web serve`.
export type Config = { api: string; token?: string };

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
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
  // A year, SameSite=Lax: the console is the only origin that reads it, and the API is called with a
  // header rather than a cookie, so there is no cross-site request to forge.
  document.cookie = `${TOKEN_COOKIE}=${encodeURIComponent(token)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
}

export function clearToken(): void {
  document.cookie = `${TOKEN_COOKIE}=; Path=/; Max-Age=0; SameSite=Lax`;
}

// config is read once from the console's own origin: the binary that served this page knows which
// API it belongs to, and a build cannot bake that in.
let configPromise: Promise<Config> | null = null;

export function loadConfig(): Promise<Config> {
  // `npm run dev` serves no config.json: a developer's console calls the constant local API instead of
  // failing to start, and the sign-in card appears because there is no session to inherit.
  const fallback: Config = { api: import.meta.env.VITE_API_URL ?? 'http://localhost:9000' };
  configPromise ??= fetch('./config.json', { headers: { Accept: 'application/json' } })
    .then(async (response) => {
      if (!response.ok) throw new Error(`config.json answered ${response.status}`);
      return (await response.json()) as Config;
    })
    .catch(() => fallback);
  return configPromise;
}

type Envelope<T> = { data?: T; meta?: { next_cursor?: string }; error?: { code: string; message: string } };

async function request<T>(
  api: string,
  method: string,
  path: string,
  body: unknown,
  override?: string,
): Promise<{ data: T; next?: string }> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  const token = override || readToken();
  if (token) headers.Authorization = `Bearer ${token}`;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  const response = await fetch(`${api}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  let envelope: Envelope<T>;
  try {
    envelope = (await response.json()) as Envelope<T>;
  } catch {
    throw new ApiError(response.status, 'unreadable', `the API answered ${response.status} without a body`);
  }

  if (!response.ok || envelope.error) {
    const error = envelope.error ?? { code: 'internal', message: `the API answered ${response.status}` };
    if (response.status === 401) clearToken();
    throw new ApiError(response.status, error.code, error.message);
  }
  return { data: envelope.data as T, next: envelope.meta?.next_cursor };
}

function query(params: Record<string, string | number | boolean | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === '' || value === false) continue;
    search.set(key, String(value));
  }
  const encoded = search.toString();
  return encoded ? `?${encoded}` : '';
}

export type ListTargets = { limit?: number; cursor?: string; q?: string; driver?: string; include_archived?: boolean };

// token overrides the cookie for one client: the request that proves a token must not depend on the
// browser having stored it a line earlier.
export function api(apiURL: string, token?: string) {
  const call = <T>(method: string, path: string, body?: unknown) =>
    request<T>(apiURL, method, path, body, token);
  return {
    whoami: () => call<Identity>('GET', '/v1/whoami').then((r) => r.data),

    targets: (filter: ListTargets = {}): Promise<Page<Target>> =>
      call<Target[]>('GET', `/v1/targets${query({ ...filter })}`).then((r) => ({
        items: r.data ?? [],
        next: r.next,
      })),

    target: (ref: string) => call<Target>('GET', `/v1/targets/${ref}`).then((r) => r.data),

    createTarget: (body: Record<string, unknown>) =>
      call<Target>('POST', '/v1/targets', body).then((r) => r.data),

    archiveTarget: (ref: string) => call<Target>('DELETE', `/v1/targets/${ref}`).then((r) => r.data),

    probeTarget: (ref: string) =>
      call<{ target: Target; capabilities: Capabilities }>('POST', `/v1/targets/${ref}/probe`).then((r) => r.data),

    capabilities: (ref: string) =>
      call<Capabilities>('GET', `/v1/targets/${ref}/capabilities`).then((r) => r.data),

    databases: (ref: string) => call<string[]>('GET', `/v1/targets/${ref}/databases`).then((r) => r.data ?? []),

    tables: (ref: string, db: string) =>
      call<Table[]>('GET', `/v1/targets/${ref}/databases/${encodeURIComponent(db)}/tables`).then(
        (r) => r.data ?? [],
      ),

    table: (ref: string, db: string, table: string) =>
      call<TableDetail>(
        'GET',
        `/v1/targets/${ref}/databases/${encodeURIComponent(db)}/tables/${encodeURIComponent(table)}`,
      ).then((r) => r.data),

    query: (ref: string, sql: string, limit?: number) =>
      call<Rows>('POST', `/v1/targets/${ref}/query`, { sql, limit }).then((r) => r.data),

    proposals: (filter: { target?: string; kind?: string; status?: string; limit?: number; cursor?: string } = {}) =>
      call<Proposal[]>('GET', `/v1/proposals${query({ ...filter })}`).then((r) => ({
        items: r.data ?? [],
        next: r.next,
      })),

    proposal: (ref: string) => call<ProposalDetail>('GET', `/v1/proposals/${ref}`).then((r) => r.data),

    propose: (targetRef: string, body: Record<string, unknown>) =>
      call<ProposalDetail>('POST', `/v1/targets/${targetRef}/proposals`, body).then((r) => r.data),

    rehearse: (ref: string, tier?: string) =>
      call<ProposalDetail>('POST', `/v1/proposals/${ref}/rehearse`, tier ? { tier } : {}).then((r) => r.data),

    decide: (ref: string, approve: boolean, note?: string) =>
      call<Proposal>('POST', `/v1/proposals/${ref}/${approve ? 'approve' : 'reject'}`, note ? { note } : {}).then(
        (r) => r.data,
      ),

    apply: (ref: string, confirmDestructive = false) =>
      call<Proposal>('POST', `/v1/proposals/${ref}/apply`, { confirm_destructive: confirmDestructive }).then(
        (r) => r.data,
      ),

    events: (ref: string) => call<JournalEntry[]>('GET', `/v1/proposals/${ref}/events`).then((r) => r.data ?? []),
  };
}

export type Api = ReturnType<typeof api>;
