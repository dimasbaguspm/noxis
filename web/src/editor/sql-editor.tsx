import Editor, { loader } from '@monaco-editor/react';
import type { Monaco } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import { KeyCode, KeyMod } from 'monaco-editor';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import { useEffect, useRef } from 'react';

// Monaco ships in the bundle, not from a CDN: a console on an air-gapped VPS must not wait for jsdelivr
// before an operator can write a SELECT. This module is imported lazily, so the list screen never
// downloads the editor.
self.MonacoEnvironment = { getWorker: () => new editorWorker() };
loader.config({ monaco });

export type Schema = { database: string; tables: { name: string; columns: string[] }[] };

const KEYWORDS = ['SELECT', 'FROM', 'WHERE', 'JOIN', 'GROUP BY', 'ORDER BY', 'LIMIT'];

// One provider for the whole page. Monaco keeps completion providers per language, globally, so a
// registration inside onMount without a dispose is a list that grows on every visit and repeats every
// table: the duplication was never in the data, it was in the registrations.
let provider: monaco.IDisposable | null = null;
let mounted = 0;
const live: { schema: Schema | null; schemas: string[] } = { schema: null, schemas: [] };

function suggestionsFor(
  monaco_: Monaco,
  model: monaco.editor.ITextModel,
  position: monaco.Position,
): monaco.languages.CompletionItem[] {
  const word = model.getWordUntilPosition(position);
  const range = {
    startLineNumber: position.lineNumber,
    endLineNumber: position.lineNumber,
    startColumn: word.startColumn,
    endColumn: word.endColumn,
  };
  const kind = monaco_.languages.CompletionItemKind;
  const seen = new Set<string>();
  const suggestions: monaco.languages.CompletionItem[] = [];

  // One entry per name and kind. Two tables with an `id` column suggest `id` once; `users` the table and
  // `users` the column stay two entries, because they are two different things to type.
  const push = (label: string, itemKind: number, detail?: string) => {
    const key = `${itemKind}:${label}`;
    if (seen.has(key)) return;
    seen.add(key);
    suggestions.push({ label, kind: itemKind, insertText: label, detail, range });
  };

  for (const name of live.schemas) {
    push(name, kind.Module, 'schema');
  }
  if (live.schema) {
    push(live.schema.database, kind.Module, 'schema');
    for (const table of live.schema.tables) {
      push(table.name, kind.Struct, `${live.schema.database}.${table.name}`);
      for (const column of table.columns) {
        push(column, kind.Field, table.name);
      }
    }
  }
  for (const keyword of KEYWORDS) {
    push(keyword, kind.Keyword);
  }
  return suggestions;
}

// register is idempotent on purpose: the first editor registers the provider, every other one reuses it,
// and the last unmount disposes it.
function register(monaco_: Monaco): monaco.IDisposable {
  provider ??= monaco_.languages.registerCompletionItemProvider('sql', {
    triggerCharacters: [' ', '.', '\n'],
    provideCompletionItems: (model, position) => ({
      suggestions: suggestionsFor(monaco_, model, position),
    }),
  });
  return provider;
}

export function SqlEditor({
  value,
  onChange,
  onRun,
  schema,
  schemas,
  height = 220,
  readOnly = false,
}: {
  value: string;
  onChange?: (next: string) => void;
  /** What Ctrl/Cmd+Enter does. A run button is a mouse gesture; the keyboard one is the habit. */
  onRun?: () => void;
  schema?: Schema | null;
  /** Every schema the server has, so a statement naming one of them completes. */
  schemas?: string[];
  height?: number;
  /** A statement somebody is reading, not writing: no cursor, no completion, still selectable. */
  readOnly?: boolean;
}) {
  // The provider reads what is in view through this module-level value, so reloading the metadata never
  // needs a second registration.
  live.schema = schema ?? null;
  live.schemas = schemas ?? [];
  const run = useRef<(() => void) | undefined>(onRun);
  run.current = onRun;

  // The last editor out disposes the provider: a page that navigates away and back must not leave one
  // behind per visit.
  useEffect(() => {
    mounted += 1;
    return () => {
      mounted -= 1;
      if (mounted === 0) {
        provider?.dispose();
        provider = null;
      }
    };
  }, []);

  return (
    <div className="overflow-hidden">
      <Editor
        height={height}
        defaultLanguage="sql"
        value={value}
        onChange={(next) => onChange?.(next ?? '')}
        onMount={(editor, instance) => {
          register(instance);
          editor.addCommand(KeyMod.CtrlCmd | KeyCode.Enter, () => run.current?.());
        }}
        options={{
          minimap: { enabled: false },
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
          fontSize: 13,
          fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
          padding: { top: 12, bottom: 12 },
          automaticLayout: true,
          renderLineHighlight: 'none',
          overviewRulerLanes: 0,
          wordWrap: 'on',
          readOnly,
        }}
      />
    </div>
  );
}
