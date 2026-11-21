import { beforeEach, describe, expect, it } from 'vitest';
import { readDraft, writeDraft } from './draft';

// The draft is a courtesy, not a record: it must never throw, and it must never come back for an
// instance it was not written for.
describe('draft', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('answers the statement it was given, per instance', () => {
    writeDraft('ref-a', 'SELECT * FROM users');

    expect(readDraft('ref-a')).toBe('SELECT * FROM users');
    expect(readDraft('ref-b')).toBeNull();
  });

  it('forgets a draft when the editor is emptied, instead of storing a blank', () => {
    writeDraft('ref-a', 'SELECT 1');
    writeDraft('ref-a', '   ');

    expect(readDraft('ref-a')).toBeNull();
  });

  it('answers nothing for an empty reference and writes nothing either', () => {
    writeDraft('', 'SELECT 1');

    expect(readDraft('')).toBeNull();
    expect(window.localStorage.length).toBe(0);
  });
});
