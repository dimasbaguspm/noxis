// The statement a person was writing belongs to the instance, not to the tab. A reload, a route change,
// a closed laptop or a detour through the changes list should not cost a query somebody was halfway
// through — and the console is exactly where a change is drafted before it becomes a proposal.
//
// localStorage is the right shelf: it never leaves the browser, it is not the journal, and a draft that
// mattered becomes a proposal, which is the product's durable copy. Storage can be unavailable (private
// mode, a full quota); that is a convenience lost, never an error, so every call here swallows it.

const prefix = 'noxis.query.';

export function readDraft(ref: string): string | null {
  if (!ref) return null;
  try {
    return window.localStorage.getItem(prefix + ref);
  } catch {
    return null;
  }
}

export function writeDraft(ref: string, sql: string): void {
  if (!ref) return;
  try {
    // Emptying the editor forgets the draft rather than storing an empty string: a stored blank would
    // replace the default statement with nothing on the next visit.
    if (sql.trim() === '') {
      window.localStorage.removeItem(prefix + ref);
      return;
    }
    window.localStorage.setItem(prefix + ref, sql);
  } catch {
    // A blocked or full storage must never stop a query from running.
  }
}
