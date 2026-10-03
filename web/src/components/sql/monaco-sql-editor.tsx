import { useEffect, useRef } from "react";
import Editor, { loader, type OnMount } from "@monaco-editor/react";
import type * as MonacoNS from "monaco-editor";
import monaco from "@/lib/monaco";
import type { Table } from "@/lib/api";

// This module is loaded lazily (see sql-editor.tsx) so monaco — a few MB of
// editor and SQL grammar — never lands in the entry bundle.
loader.config({ monaco });

export interface MonacoSqlEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height: string;
  tables: Table[];
  placeholder?: string;
}

const KEYWORDS = [
  "SELECT", "FROM", "WHERE", "JOIN", "LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "GROUP BY", "ORDER BY",
  "LIMIT", "OFFSET", "HAVING", "INSERT INTO", "VALUES", "UPDATE", "SET", "DELETE FROM", "CREATE TABLE",
  "ALTER TABLE", "DROP TABLE", "TRUNCATE", "ANALYZE", "EXPLAIN", "WITH", "RETURNING", "ON CONFLICT",
];

export default function MonacoSqlEditor({
  value,
  onChange,
  readOnly,
  height,
  tables,
  placeholder,
}: MonacoSqlEditorProps) {
  const completion = useRef<MonacoNS.IDisposable | undefined>(undefined);

  // schema-aware completion: tables in the outer scope, their columns after
  // FROM/JOIN/UPDATE/INTO
  const onMount: OnMount = () => {
    completion.current?.dispose();
    completion.current = monaco.languages.registerCompletionItemProvider("sql", {
      provideCompletionItems: (model, position) => {
        const line = model.getValueInRange({
          startLineNumber: position.lineNumber,
          startColumn: 1,
          endLineNumber: position.lineNumber,
          endColumn: position.column,
        });
        const word = model.getWordUntilPosition(position);
        const range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn,
        };
        const keyword = (label: string, kind: MonacoNS.languages.CompletionItemKind) => ({
          label,
          kind,
          insertText: label,
          range,
        });

        const after = /(?:from|join|update|into)\s+([\w.]+)\s+[\w]*$/i.exec(line);
        if (after) {
          const [, tableName] = after;
          const table = tables.find(
            (t) =>
              t.name.toLowerCase() === tableName.toLowerCase() ||
              `${t.schema}.${t.name}`.toLowerCase() === tableName.toLowerCase(),
          );
          if (table) {
            return {
              suggestions: table.columns.map((column) =>
                keyword(column.name, monaco.languages.CompletionItemKind.Field),
              ),
            };
          }
        }

        return {
          suggestions: [
            ...tables.map((t) => keyword(`${t.schema}.${t.name}`, monaco.languages.CompletionItemKind.Struct)),
            ...KEYWORDS.map((k) => keyword(k, monaco.languages.CompletionItemKind.Keyword)),
          ],
        };
      },
    });
  };

  useEffect(() => () => completion.current?.dispose(), []);

  return (
    <Editor
      language="sql"
      value={value}
      height={height}
      theme="vs-dark"
      onMount={onMount}
      onChange={(next) => onChange?.(next ?? "")}
      options={{
        readOnly,
        minimap: { enabled: false },
        fontSize: 12,
        lineNumbers: readOnly ? "off" : "on",
        scrollBeyondLastLine: false,
        wordWrap: "on",
        tabSize: 2,
        automaticLayout: true,
        placeholder,
        padding: { top: 8, bottom: 8 },
      }}
    />
  );
}
