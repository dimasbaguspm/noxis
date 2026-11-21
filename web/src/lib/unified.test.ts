import { describe, expect, it } from 'vitest';
import { splitUnified } from './unified';

// The rehearsal records a unified diff and the console shows it as two sides. Getting this wrong is not
// cosmetic: a reviewer reads the right-hand column to decide.
describe('splitUnified', () => {
  const diff = [
    '--- a/users',
    '+++ b/users',
    '@@ -1,3 +1,3 @@',
    ' CREATE TABLE `users` (',
    '-  `note` TEXT,',
    '+  `note` TEXT NOT NULL,',
    ' )',
  ].join('\n');

  it('puts a removed line on the left and an added line on the right', () => {
    const { original, modified } = splitUnified(diff);

    expect(original).toContain('`note` TEXT,');
    expect(original).not.toContain('NOT NULL');

    expect(modified).toContain('`note` TEXT NOT NULL,');
    expect(modified).not.toContain('`note` TEXT,');
  });

  it('keeps a context line on both sides', () => {
    const { original, modified } = splitUnified(diff);

    expect(original).toContain('CREATE TABLE `users`');
    expect(modified).toContain('CREATE TABLE `users`');
  });

  it('drops the file headers, which name files this diff does not have', () => {
    const { original, modified } = splitUnified(diff);

    expect(original).not.toContain('--- a/users');
    expect(modified).not.toContain('+++ b/users');
  });

  it('answers two empty sides for an empty diff', () => {
    expect(splitUnified('')).toEqual({ original: '', modified: '' });
  });
});
