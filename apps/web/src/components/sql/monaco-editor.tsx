import { useEffect, useRef } from "react";
import Editor, { loader, type OnMount } from "@monaco-editor/react";
import type * as MonacoNS from "monaco-editor";
import monaco, { MONGO_LANGUAGE } from "@/lib/monaco";
import type { Table } from "@/lib/api";
// This module is loaded lazily (see sql-editor.tsx / json-editor.tsx) so monaco — a few MB of editor and grammars — never lands in the entry bundle.
loader.config({ monaco });

export interface MonacoEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height: string;
  language: "sql" | "mongo";
  // SQL: the tables to complete against. JSON: the document fields to suggest.
  tables?: Table[];
  fields?: string[];
  placeholder?: string;
  // onRun is bound to Ctrl/Cmd+Enter in the editor.
  onRun?: () => void;
}

// MongoDB query and update operators: the vocabulary a filter or value document is written in.
const MONGO_OPERATORS = [
  "$eq",
  "$ne",
  "$gt",
  "$gte",
  "$lt",
  "$lte",
  "$in",
  "$nin",
  "$exists",
  "$type",
  "$regex",
  "$all",
  "$elemMatch",
  "$size",
  "$and",
  "$or",
  "$nor",
  "$not",
  "$expr",
  "$mod",
  "$text",
  "$where",
  "$set",
  "$unset",
  "$inc",
  "$mul",
  "$min",
  "$max",
  "$rename",
  "$push",
  "$pull",
  "$pop",
  "$addToSet",
  "$currentDate",
  "$setOnInsert",
  "$each",
  "$slice",
  "$sort",
  "$position",
  "$match",
  "$group",
  "$project",
  "$lookup",
  "$unwind",
  "$limit",
  "$skip",
  "$count",
  "$sum",
  "$avg",
];

const KEYWORDS = [
  "SELECT",
  "FROM",
  "WHERE",
  "JOIN",
  "LEFT JOIN",
  "RIGHT JOIN",
  "INNER JOIN",
  "GROUP BY",
  "ORDER BY",
  "LIMIT",
  "OFFSET",
  "HAVING",
  "INSERT INTO",
  "VALUES",
  "UPDATE",
  "SET",
  "DELETE FROM",
  "CREATE TABLE",
  "ALTER TABLE",
  "DROP TABLE",
  "TRUNCATE",
  "ANALYZE",
  "EXPLAIN",
  "WITH",
  "RETURNING",
  "ON CONFLICT",
];

export default function MonacoEditor({
  value,
  onChange,
  readOnly,
  height,
  language,
  tables = [],
  fields = [],
  placeholder,
  onRun,
}: MonacoEditorProps) {
  const completion = useRef<MonacoNS.IDisposable | undefined>(undefined);
  // the run callback changes with the query state; read it through a ref
  const onRunRef = useRef(onRun);
  onRunRef.current = onRun;
  // the provider lives for the editor's lifetime and the schema arrives asynchronously, so it reads the latest tables/fields through a ref
  const tablesRef = useRef(tables);
  tablesRef.current = tables;
  const fieldsRef = useRef(fields);
  fieldsRef.current = fields;

  // Completion per language: SQL suggests tables and their columns after FROM/JOIN; JSON suggests MongoDB operators and the collection's fields.
  const onMount: OnMount = (editor) => {
    completion.current?.dispose();
    // Ctrl/Cmd+Enter runs the query instead of inserting a new line
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, () => onRunRef.current?.());
    if (language === MONGO_LANGUAGE) {
      completion.current = monaco.languages.registerCompletionItemProvider(MONGO_LANGUAGE, {
        triggerCharacters: ["$", '"', "{", ","],
        provideCompletionItems: (model, position) => {
          const word = model.getWordUntilPosition(position);
          const range = {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: word.startColumn,
            endColumn: word.endColumn,
          };
          const item = (label: string, kind: MonacoNS.languages.CompletionItemKind) => ({
            label,
            kind,
            insertText: label,
            range,
          });
          const line = model.getValueInRange({
            startLineNumber: position.lineNumber,
            startColumn: 1,
            endLineNumber: position.lineNumber,
            endColumn: position.column,
          });
          // after a "$" only operators make sense
          if (/\$[\w]*$/.test(line)) {
            return {
              suggestions: MONGO_OPERATORS.map((op) =>
                item(op, monaco.languages.CompletionItemKind.Operator),
              ),
            };
          }
          return {
            suggestions: [
              ...fieldsRef.current.map((field) =>
                item(field, monaco.languages.CompletionItemKind.Field),
              ),
              ...MONGO_OPERATORS.map((op) =>
                item(op, monaco.languages.CompletionItemKind.Operator),
              ),
            ],
          };
        },
      });
      return;
    }
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
          const table = tablesRef.current.find(
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
            ...tablesRef.current.map((t) =>
              keyword(`${t.schema}.${t.name}`, monaco.languages.CompletionItemKind.Struct),
            ),
            ...KEYWORDS.map((k) => keyword(k, monaco.languages.CompletionItemKind.Keyword)),
          ],
        };
      },
    });
  };

  useEffect(() => () => completion.current?.dispose(), []);

  return (
    <Editor
      language={language}
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
