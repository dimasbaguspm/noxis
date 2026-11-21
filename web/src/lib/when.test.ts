import { describe, expect, it } from 'vitest';
import dayjs from 'dayjs';
import { formatWhen } from './when';

// A timestamp is the fact a reviewer checks first: when did this land, and when was it probed.
describe('formatWhen', () => {
  it('answers nothing for a value the server did not send', () => {
    expect(formatWhen()).toBe('');
    expect(formatWhen(null)).toBe('');
    expect(formatWhen('')).toBe('');
  });

  it('answers the original string when it is not a date, rather than "Invalid Date"', () => {
    expect(formatWhen('not-a-date')).toBe('not-a-date');
  });

  it('reads as a relative time inside a day', () => {
    const twoHoursAgo = dayjs().subtract(2, 'hour').toISOString();

    expect(formatWhen(twoHoursAgo)).toContain('ago');
  });

  it('reads as an absolute time after a day', () => {
    const threeDaysAgo = dayjs().subtract(3, 'day').toISOString();

    expect(formatWhen(threeDaysAgo)).toMatch(/^\d{1,2} [A-Z][a-z]{2} \d{4}, \d{2}:\d{2}$/);
  });
});
