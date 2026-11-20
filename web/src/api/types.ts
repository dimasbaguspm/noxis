// The wire shapes, as the API documents them at /openapi.json. They are `type` aliases rather than a
// generated client because the console uses a handful of routes; the day it uses all of them, this
// file is replaced by a generator.

export type Target = {
  ref: string;
  name: string;
  driver: 'mysql' | 'mariadb';
  host: string;
  port: number;
  database: string;
  username: string;
  tls_mode: string;
  options?: Record<string, string>;
  has_password: boolean;
  created_by: string;
  server_version?: string;
  last_probe_at?: string;
  last_probe_status?: string;
  archived_at?: string;
  created_at: string;
  updated_at: string;
};

export type Capabilities = {
  driver: string;
  version: string;
  rehearsal: boolean;
  supports_ddl: boolean;
  supports_dml: boolean;
  background_apply: boolean;
  explain: boolean;
};

export type Proposal = {
  ref: string;
  target_ref: string;
  title: string;
  query: string;
  rollback?: string;
  note?: string;
  kind: 'ddl' | 'dml';
  status: string;
  tier: string;
  proposed_by: string;
  decided_by?: string;
  decided_at?: string;
  decided_note?: string;
  applied_by?: string;
  applied_at?: string;
  rows_estimated?: number;
  failure?: string;
  irreversible?: boolean;
  algorithm?: string;
  rows_affected?: number;
  started_at?: string;
  progress_stage?: string;
  progress_percent?: number;
  created_at: string;
  updated_at: string;
};

export type Rehearsal = {
  ref: string;
  proposal_ref: string;
  tier: string;
  status: string;
  irreversible: boolean;
  note?: string;
  error?: string;
  schema_diff?: string;
  row_diff?: string;
  algorithm?: string;
  rows_affected?: number;
  duration_ms?: number;
  plan?: string;
  created_at?: string;
  finished_at?: string;
};

export type ProposalDetail = Proposal & { rehearsal?: Rehearsal };

export type JournalEntry = {
  ref: string;
  subject_ref: string;
  subject_kind: string;
  kind: string;
  actor: string;
  detail: Record<string, string>;
  created_at: string;
};

export type Rows = {
  columns: { name: string }[];
  values: (string | null)[][];
  row_count: number;
  truncated: boolean;
};

export type Table = { name: string; rows_estimated?: number };

export type TableDetail = {
  name: string;
  columns: { name: string; type: string; nullable?: boolean; key?: string }[];
  indexes?: { name: string; columns: string[]; unique?: boolean }[];
  ddl?: string;
};

export type Identity = { name: string; scopes: string[] };

export type Page<T> = { items: T[]; next?: string };
