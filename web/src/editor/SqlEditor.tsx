import Editor, { loader } from '@monaco-editor/react';
import type { Monaco } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import { useRef } from 'react';

// Monaco ships in the bundle, not from a CDN: a console on an air-gapped VPS must not wait for
// jsdelivr to answer before an operator can write a SELECT. This module is imported lazily by the
// instance page, so the list screen never downloads the editor.
self.MonacoEnvironment = { getWorker: () => new editorWorker() };
loader.config({ monaco });

// The editor is Monaco, and it knows the database: the completion provider reads the target's own
// schema, so `FROM ` offers tables that exist and `SELECT ` offers their columns. When metadata has
// not arrived yet it still completes keywords — a suggestion that is merely syntax is better than an
// editor that argues.

export type Schema = {
  database: string;
  tables: { name: string; columns: string[] }[];
};

const KEYWORDS = ['SELECT', 'FROM', 'WHERE', 'JOIN', 'GROUP BY', 'ORDER BY', 'LIMIT', 'UPDATE', 'SET'];

function provider(monaco: Monaco, schema: { current: Schema | null }) {
  return monaco.languages.registerCompletionItemProvider('sql', {
    triggerCharacters: [' ', '.', '\n'],
    provideCompletionItems: (model, position) => {
      const word = model.getWordUntilPosition(position);
      const range = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      };
      const suggestions: {
        label: string;
        kind: number;
        insertText: string;
        detail?: string;
        range: typeof range;
      }[] = [];

      const known = schema.current;
      if (known) {
        for (const table of known.tables) {
          suggestions.push({
            label: table.name,
            kind: monaco.languages.CompletionItemKind.Struct,
            insertText: table.name,
            detail: `${known.database}.${table.name}`,
            range,
          });
          for (const column of table.columns) {
            suggestions.push({
              label: column,
              kind: monaco.languages.CompletionItemKind.Field,
              insertText: column,
              detail: table.name,
              range,
            });
          }
        }
      }
      for (const keyword of KEYWORDS) {
        suggestions.push({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          insertText: keyword,
          range,
        });
      }
      return { suggestions };
    },
  });
}

export function SqlEditor({
  value,
  onChange,
  schema,
  height = 220,
  readOnly,
}: {
  value: string;
  onChange?: (next: string) => void;
  schema?: Schema | null;
  height?: number;
  readOnly?: boolean;
}) {
  // The provider is registered once per mount and reads the schema through a ref, so a reload of the
  // metadata never needs a second registration.
  const current = useRef<Schema | null>(schema ?? null);
  current.current = schema ?? null;

  return (
    <div className="noxis-editor">
      <Editor
        height={height}
        defaultLanguage="sql"
        value={value}
        onChange={(next) => onChange?.(next ?? '')}
        onMount={(_, monaco) => {
          provider(monaco, current);
        }}
        options={{
          readOnly,
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
        }}
      />
    </div>
  );
}
