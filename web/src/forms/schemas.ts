import { z } from 'zod';

// One schema per form, so the rules live in one place and the field renders the message the schema
// produced. The enums mirror what the server accepts: a value the API would refuse is not offered.

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

// The queues filter the page the API returned; there is no `q` for proposals, and filtering what is
// on screen is honest where pretending to search the whole queue would not be.
export const statuses = ['verified', 'approved', 'applied', 'failed', 'rejected'] as const;
export type Status = (typeof statuses)[number];
