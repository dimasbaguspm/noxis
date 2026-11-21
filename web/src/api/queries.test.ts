import { describe, expect, it } from 'vitest';
import { clientFor } from './queries';

// A console served by the API is told the empty origin, which means "the origin you loaded this page
// from". It is falsy, so a truthiness check somewhere would leave every query disabled and draw an empty
// list on a server full of rows — the failure that shipped once already.
describe('clientFor', () => {
  it('builds a client for the empty origin', () => {
    const client = clientFor({ api: '' });
    expect(client).not.toBeNull();
    expect(client?.defaults.baseURL).toBe('');
    expect(client?.getUri({ url: '/v1/targets' })).toBe('/v1/targets');
  });

  it('builds a client for a named origin', () => {
    const client = clientFor({ api: 'https://api.example.com' });
    expect(client?.getUri({ url: '/v1/targets' })).toBe('https://api.example.com/v1/targets');
  });

  // Told nothing, the console calls nothing: a query that fires at an unknown host is worse than a page
  // that waits.
  it('builds nothing when the console was told no API', () => {
    expect(clientFor(null)).toBeNull();
  });
});
