import { z } from 'zod';

// One schema per form. The rules live here and the field renders the message the schema produced; the
// enums are the values the API accepts, so a choice the server would refuse is not on screen.

export const tokenSchema = z.object({
  token: z.string().trim().min(1, 'a token is required'),
});
export type TokenValues = z.infer<typeof tokenSchema>;

export const targetSchema = z.object({
  name: z.string().trim().min(1, 'a name is required').max(64, '64 characters at most'),
  driver: z.enum(['mysql', 'mariadb']),
  host: z.string().trim().min(1, 'a host is required'),
  port: z.coerce.number().int().min(1, '1-65535').max(65535, '1-65535'),
  database: z.string().trim().min(1, 'a database is required'),
  username: z.string().trim().min(1, 'a username is required'),
  password: z.string().optional(),
  tls_mode: z.enum(['preferred', 'skip-verify', 'verify']),
});
export type TargetValues = z.infer<typeof targetSchema>;

export const targetDefaults: TargetValues = {
  name: '',
  driver: 'mysql',
  host: '',
  port: 3306,
  database: '',
  username: '',
  password: '',
  tls_mode: 'preferred',
};

export const driverOptions = [
  { value: 'mysql', label: 'MySQL' },
  { value: 'mariadb', label: 'MariaDB' },
];

export const tlsOptions = [
  { value: 'preferred', label: 'Preferred' },
  { value: 'verify', label: 'Verify' },
  { value: 'skip-verify', label: 'Skip verify' },
];

export const proposalSchema = z.object({
  title: z.string().trim().min(1, 'a title is required').max(120, '120 characters at most'),
  query: z.string().trim().min(1, 'the statement is required'),
  rollback: z.string().optional(),
});
export type ProposalValues = z.infer<typeof proposalSchema>;

export const proposalDefaults: ProposalValues = {
  title: '',
  query: '',
  rollback: '',
};

export const statuses = ['verified', 'approved', 'applied', 'failed'] as const;
