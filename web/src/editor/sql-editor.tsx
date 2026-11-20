import Editor, { loader } from '@monaco-editor/react';
import type { Monaco } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import { useRef } from 'react';

// Monaco ships in the bundle, not from a CDN: a console on an air-gapped VPS must not wait for jsdelivr
// before an operator can write a SELECT. This module is imported lazily, so the list screen never
// downloads the editor.
self.MonacoEnvironment = { getWorker: () => new editorWorker() };
loader.config({ monaco });

export type Schema = { database: string; tables: { name: string; columns: string[] }[] };

const KEYWORDS = ['SELECT', 'FROM', 'WHERE', 'JOIN', 'GROUP BY', 'ORDER BY', 'LIMIT'];

function completion(monaco_: Monaco, schema: { current: Schema | null }) {
  return monaco_.languages.registerCompletionItemProvider('sql', {
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
            kind: monaco_.languages.CompletionItemKind.Struct,
            insertText: table.name,
            detail: `${known.database}.${table.name}`,
            range,
          });
          for (const column of table.columns) {
            suggestions.push({
              label: column,
              kind: monaco_.languages.CompletionItemKind.Field,
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
          kind: monaco_.languages.CompletionItemKind.Keyword,
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
}: {
  value: string;
  onChange?: (next: string) => void;
  schema?: Schema | null;
  height?: number;
}) {
  // The provider is registered once per mount and reads the schema through a ref, so reloading the
  // metadata never needs a second registration.
  const current = useRef<Schema | null>(schema ?? null);
  current.current = schema ?? null;

  return (
    <div className="overflow-hidden rounded-lg border border-line">
      <Editor
        height={height}
        defaultLanguage="sql"
        value={value}
        onChange={(next) => onChange?.(next ?? '')}
        onMount={(_, instance) => {
          completion(instance, current);
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
        }}
      />
    </div>
  );
}
