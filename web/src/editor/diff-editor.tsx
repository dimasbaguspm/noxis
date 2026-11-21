import { DiffEditor, loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import { useEffect, useState } from 'react';

// The schema diff is the evidence a reviewer decides from, and it was rendered in a plain read-only
// editor: one column, no gutter, no sense of what was added to what. Monaco ships a diff editor and the
// console already bundles Monaco, so the same rule applies — in the bundle, never a CDN, imported
// lazily so a page with no diff never downloads it.
self.MonacoEnvironment = { getWorker: () => new editorWorker() };
loader.config({ monaco });

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

/** wide is the breakpoint the page's two-column layout uses; a diff follows the same rule. */
function useWideLayout(): boolean {
  const [wide, setWide] = useState(() => window.matchMedia('(min-width: 1024px)').matches);
  useEffect(() => {
    const query = window.matchMedia('(min-width: 1024px)');
    const listen = (event: MediaQueryListEvent) => setWide(event.matches);
    query.addEventListener('change', listen);
    return () => query.removeEventListener('change', listen);
  }, []);
  return wide;
}

export function SchemaDiff({ diff, height }: { diff: string; height?: number }) {
  const wide = useWideLayout();
  const { original, modified } = splitUnified(diff);
  // A four-line diff in a 240px box is mostly empty space: the box is as tall as the diff, up to a cap.
  const lines = Math.max(original.split('\n').length, modified.split('\n').length);

  return (
    <div
      className="overflow-hidden rounded-lg border border-line"
      style={{ height: height ?? Math.min(360, Math.max(90, lines * 20 + 16)) }}
    >
      <DiffEditor
        height="100%"
        original={original}
        modified={modified}
        language="sql"
        theme="vs"
        options={{
          readOnly: true,
          originalEditable: false,
          renderSideBySide: wide,
          minimap: { enabled: false },
          renderOverviewRuler: false,
          scrollBeyondLastLine: false,
          fontSize: 13,
          automaticLayout: true,
        }}
      />
    </div>
  );
}
