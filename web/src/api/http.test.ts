import { describe, expect, it } from 'vitest';
import { configFrom } from './http';

// The answer a build gets from the binary that served it. It is checked rather than trusted: a console
// that reads an unexpected document and calls nothing is a page that reports a network error nobody can
// explain, so a wrong answer falls back to the origin the page came from.
describe('configFrom', () => {
  const fallback = { api: '/v1' };

  it('takes the origin the server named', () => {
    expect(configFrom({ api: 'https://api.example.com' }, fallback)).toEqual({
      api: 'https://api.example.com',
    });
  });

  // An empty origin is the normal answer: the page and the API are the same origin, so there is no
  // address to name and the calls go where the page came from.
  it('takes an empty origin as this origin', () => {
    expect(configFrom({ api: '' }, { api: 'https://elsewhere.example.com' })).toEqual({ api: '' });
  });

  it('falls back when the answer is not a config', () => {
    for (const answer of [null, undefined, {}, { api: 42 }, '<!doctype html>']) {
      expect(configFrom(answer, fallback)).toEqual(fallback);
    }
  });

  it('never carries a token, whatever the server sends', () => {
    const config = configFrom({ api: '', token: 'smuggled' }, fallback);
    expect(config).toEqual({ api: '' });
    expect('token' in config).toBe(false);
  });
});
