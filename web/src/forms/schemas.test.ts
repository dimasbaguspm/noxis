import { describe, expect, it } from 'vitest';
import { connectionDefaults, connectionSchema, targetSchema } from './schemas';

// The schemas are the console's half of a contract the API also enforces: a field the server would refuse
// is refused here, before a request is made.
describe('targetSchema', () => {
  const complete = {
    name: 'prod',
    driver: 'mysql' as const,
    host: 'db.internal',
    port: 3306,
    database: 'app',
    username: 'noxis',
    password: 'typed-once',
    tls_mode: 'preferred' as const,
  };

  it('accepts a complete registration', () => {
    expect(targetSchema.safeParse(complete).success).toBe(true);
  });

  it('refuses a host the server would refuse', () => {
    const result = targetSchema.safeParse({ ...complete, host: '  ' });

    expect(result.success).toBe(false);
  });

  it('refuses a driver this build does not speak', () => {
    expect(targetSchema.safeParse({ ...complete, driver: 'postgres' }).success).toBe(false);
  });

  it('refuses a port outside 1-65535', () => {
    expect(targetSchema.safeParse({ ...complete, port: 70000 }).success).toBe(false);
  });
});

describe('connectionSchema', () => {
  it('accepts an empty password, which means "keep the stored one"', () => {
    const result = connectionSchema.safeParse({
      name: 'prod',
      username: 'noxis',
      password: '',
      tls_mode: 'verify',
    });

    expect(result.success).toBe(true);
  });

  it('seeds a row without its password, because the server never reads one back', () => {
    const values = connectionDefaults({ name: 'prod', username: 'noxis', tls_mode: 'skip-verify' });

    expect(values.password).toBe('');
    expect(values.tls_mode).toBe('skip-verify');
  });

  it('falls back to the default TLS mode for a value a row could not have', () => {
    const values = connectionDefaults({ name: 'prod', username: 'noxis', tls_mode: 'nonsense' });

    expect(values.tls_mode).toBe('preferred');
  });
});
