import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

// One place turns a wire timestamp into something a person reads. The API sends RFC 3339 in UTC and
// every page printed it raw: a reviewer had to decode `2026-11-21T01:12:44.815Z` to answer "how long
// ago was that". Relative within a day, absolute after it, and the original string when the value is
// not a date at all, because a console that prints "Invalid Date" is worse than one that prints what
// it was given.
dayjs.extend(relativeTime);

const ABSOLUTE = 'D MMM YYYY, HH:mm';

export function formatWhen(value?: string | null): string {
  if (!value) return '';
  const at = dayjs(value);
  if (!at.isValid()) return String(value);
  return Math.abs(dayjs().diff(at, 'hour')) < 24 ? at.fromNow() : at.format(ABSOLUTE);
}
