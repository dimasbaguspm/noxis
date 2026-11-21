import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { createClient, params, unwrap, unwrapPage } from './http';
import { useSession } from './session-context';
import type { ListTargets } from './http';
import type { Page } from './http';
import type {
  Capabilities,
  JournalEntry,
  Proposal,
  ProposalDetail,
  Rows,
  Table,
  TableDetail,
  Target,
  TestAnswer,
} from './types';

// Every read and write the console makes, as a hook. A screen says which data it needs and which
// action it runs; nothing in a page touches axios, a URL, or a cache key.

export const keys = {
  session: ['whoami'] as const,
  targets: (filter: ListTargets) => ['targets', filter] as const,
  target: (ref: string) => ['target', ref] as const,
  capabilities: (ref: string) => ['capabilities', ref] as const,
  tables: (ref: string, db: string) => ['tables', ref, db] as const,
  databases: (ref: string) => ['databases', ref] as const,
  rows: (ref: string, db: string, table: string) => ['rows', ref, db, table] as const,
  proposals: (filter: Record<string, string | number | undefined>) =>
    ['proposals', filter] as const,
  proposal: (ref: string) => ['proposal', ref] as const,
  events: (ref: string) => ['events', ref] as const,
};

function useClient() {
  const { api } = useSession();
  return useMemo(() => (api ? createClient(api) : null), [api]);
}

/** useTargets is the bug that made this file: one infinite query, one search term, no double fetch. */
export function useTargets(term: string, includeArchived = false) {
  const client = useClient();
  const filter: ListTargets = { q: term, include_archived: includeArchived, limit: 24 };
  return useInfiniteQuery({
    queryKey: keys.targets(filter),
    enabled: Boolean(client),
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      unwrapPage<Target>(client!.get('/v1/targets', params({ ...filter, cursor: pageParam }))),
    getNextPageParam: (last: Page<Target>) => last.next,
  });
}

export function useTarget(ref: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.target(ref),
    enabled: Boolean(client && ref),
    queryFn: () => unwrap<Target>(client!.get(`/v1/targets/${ref}`)),
  });
}

export function useCapabilities(ref: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.capabilities(ref),
    enabled: Boolean(client && ref),
    queryFn: () => unwrap<Capabilities>(client!.get(`/v1/targets/${ref}/capabilities`)),
  });
}

/** useSchema is what the editor completes from: the tables, then the columns of each one. */
export function useSchema(ref: string, database: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.tables(ref, database),
    enabled: Boolean(client && ref && database),
    staleTime: 60_000,
    queryFn: async () => {
      const tables = await unwrap<Table[]>(
        client!.get(`/v1/targets/${ref}/databases/${encodeURIComponent(database)}/tables`),
      );
      const details = await Promise.all(
        tables
          .slice(0, 200)
          .map((table) =>
            unwrap<TableDetail>(
              client!.get(
                `/v1/targets/${ref}/databases/${encodeURIComponent(database)}/tables/${encodeURIComponent(table.name)}`,
              ),
            ),
          ),
      );
      return {
        database,
        tables: details.map((detail) => ({
          name: detail.name ?? '',
          columns: (detail.columns ?? []).map((column) => column.name),
        })),
      };
    },
  });
}

export function useProposals(filter: {
  target?: string;
  kind?: 'dml' | 'ddl';
  status?: string;
  limit?: number;
}) {
  const client = useClient();
  return useQuery({
    queryKey: keys.proposals(filter),
    enabled: Boolean(client),
    queryFn: () => unwrapPage<Proposal>(client!.get('/v1/proposals', params({ ...filter }))),
  });
}

/** useTestConnection dials a configuration nobody stored yet. It writes nothing, so the form can ask
 * before it decides to register. */
export function useTestConnection() {
  const client = useClient();
  return useMutation({
    mutationFn: (input: Record<string, unknown>) =>
      unwrap<TestAnswer>(client!.post('/v1/targets/test', input)),
  });
}

export function useProposal(ref: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.proposal(ref),
    enabled: Boolean(client && ref),
    queryFn: () => unwrap<ProposalDetail>(client!.get(`/v1/proposals/${ref}`)),
  });
}

export function useEvents(ref: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.events(ref),
    enabled: Boolean(client && ref),
    queryFn: () => unwrap<JournalEntry[]>(client!.get(`/v1/proposals/${ref}/events`)),
  });
}

export function useRunQuery(ref: string) {
  const client = useClient();
  return useMutation({
    mutationFn: (input: { sql: string; limit?: number }) =>
      unwrap<Rows>(client!.post(`/v1/targets/${ref}/query`, input)),
  });
}

export function useCreateTarget() {
  const client = useClient();
  const cache = useQueryClient();
  return useMutation({
    mutationFn: (input: Record<string, unknown>) =>
      unwrap<Target>(client!.post('/v1/targets', input)),
    onSuccess: () => cache.invalidateQueries({ queryKey: ['targets'] }),
  });
}

export function useArchiveTarget() {
  const client = useClient();
  const cache = useQueryClient();
  return useMutation({
    mutationFn: (ref: string) => unwrap<Target>(client!.delete(`/v1/targets/${ref}`)),
    onSuccess: () => cache.invalidateQueries({ queryKey: ['targets'] }),
  });
}

/** useTables is the light list: names the API can answer without describing each table. */
export function useTables(ref: string, database: string) {
  const client = useClient();
  return useQuery({
    queryKey: ['table-list', ref, database],
    enabled: Boolean(client && ref && database),
    staleTime: 60_000,
    queryFn: () =>
      unwrap<Table[]>(
        client!.get(`/v1/targets/${ref}/databases/${encodeURIComponent(database)}/tables`),
      ),
  });
}

/** useDatabases is every schema the server has. MySQL and MariaDB call a schema a database, and a target
 * only registers one of them, so the pickers ask instead of assuming. */
export function useDatabases(ref: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.databases(ref),
    enabled: Boolean(client && ref),
    staleTime: 60_000,
    queryFn: () => unwrap<string[]>(client!.get(`/v1/targets/${ref}/databases`)),
  });
}

/** useCreateProposal files one change against a target, at the tier the proposer chose. */
export function useCreateProposal(ref: string) {
  const client = useClient();
  const cache = useQueryClient();
  return useMutation({
    mutationFn: (input: { title: string; query: string; rollback?: string; note?: string }) =>
      unwrap<Proposal>(client!.post(`/v1/targets/${ref}/proposals`, input)),
    onSuccess: () => cache.invalidateQueries({ queryKey: ['proposals'] }),
  });
}

export function useProbeTarget() {
  const client = useClient();
  const cache = useQueryClient();
  return useMutation({
    mutationFn: (ref: string) =>
      unwrap<{ target: Target; capabilities: Capabilities }>(
        client!.post(`/v1/targets/${ref}/probe`),
      ),
    onSuccess: (_data, ref) => {
      void cache.invalidateQueries({ queryKey: keys.target(ref) });
      void cache.invalidateQueries({ queryKey: ['targets'] });
    },
  });
}

/** useProposalActions is one hook for the four things a reviewer does, so a page names the verb. */
export function useProposalActions(ref: string) {
  const client = useClient();
  const cache = useQueryClient();
  const refresh = () => {
    void cache.invalidateQueries({ queryKey: keys.proposal(ref) });
    void cache.invalidateQueries({ queryKey: keys.events(ref) });
    void cache.invalidateQueries({ queryKey: ['proposals'] });
  };

  return {
    rehearse: useMutation({
      mutationFn: () => unwrap<ProposalDetail>(client!.post(`/v1/proposals/${ref}/rehearse`, {})),
      onSuccess: refresh,
    }),
    decide: useMutation({
      mutationFn: (input: { approve: boolean; note?: string }) =>
        unwrap<Proposal>(
          client!.post(`/v1/proposals/${ref}/${input.approve ? 'approve' : 'reject'}`, {
            note: input.note,
          }),
        ),
      onSuccess: refresh,
    }),
    apply: useMutation({
      mutationFn: (confirmDestructive: boolean) =>
        unwrap<Proposal>(
          client!.post(`/v1/proposals/${ref}/apply`, {
            confirm_destructive: confirmDestructive,
          }),
        ),
      onSuccess: refresh,
    }),
  };
}

/** The read route's ceiling, named once: a big table answers instead of hanging, and the count says so. */
const ROW_LIMIT = 200;

/** A name is quoted because `order` and `my-table` are legal names and a bare one is not SQL. */
function qualified(database: string, table: string): string {
  return `\`${database.replaceAll('`', '``')}\`.\`${table.replaceAll('`', '``')}\``;
}

/** useRows is the first page of one table, read through the route the query tab uses: one path to the
 * engine, and the same ceiling and the same truncation answer. */
export function useRows(ref: string, database: string, table: string) {
  const client = useClient();
  return useQuery({
    queryKey: keys.rows(ref, database, table),
    enabled: Boolean(client && ref && database && table),
    queryFn: () =>
      unwrap<Rows>(
        client!.post(`/v1/targets/${ref}/query`, {
          sql: `SELECT * FROM ${qualified(database, table)} LIMIT ${ROW_LIMIT}`,
        }),
      ),
  });
}

/** useUpdateTarget is a PATCH: a field that is absent is a field nobody asked to change, which is what
 * lets an empty password leave the stored one alone. */
export function useUpdateTarget(ref: string) {
  const client = useClient();
  const cache = useQueryClient();
  return useMutation({
    mutationFn: (input: Record<string, unknown>) =>
      unwrap<Target>(client!.patch(`/v1/targets/${ref}`, input)),
    onSuccess: () => {
      void cache.invalidateQueries({ queryKey: keys.target(ref) });
      void cache.invalidateQueries({ queryKey: ['targets'] });
    },
  });
}
