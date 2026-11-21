// A unified diff, split into its two sides. It lives here rather than beside the editor for one reason:
// it is pure, and a pure function can be tested without a browser, a canvas or Monaco.

/**
 * splitUnified turns the unified diff the rehearsal records into its two sides. A line that starts with
 * `+` exists only after the change, `-` only before it, and anything else is in both. The header lines
 * are dropped: they name files, and this diff names tables in its own context lines.
 */
export function splitUnified(diff: string): { original: string; modified: string } {
  const before: string[] = [];
  const after: string[] = [];
  for (const line of diff.split('\n')) {
    if (line.startsWith('+++') || line.startsWith('---')) continue;
    if (line.startsWith('+')) {
      after.push(line.slice(1));
      continue;
    }
    if (line.startsWith('-')) {
      before.push(line.slice(1));
      continue;
    }
    const shared = line.startsWith(' ') ? line.slice(1) : line;
    before.push(shared);
    after.push(shared);
  }
  return { original: before.join('\n'), modified: after.join('\n') };
}
