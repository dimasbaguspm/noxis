import { Suspense, lazy } from "react";
import type { Table } from "@/lib/api";

const MonacoEditor = lazy(() => import("@/components/sql/monaco-editor"));

export interface SqlEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: string;
  tables?: Table[];
  placeholder?: string;
  // onRun is bound to Ctrl/Cmd+Enter (monaco, and the loading fallback).
  onRun?: () => void;
}

// SqlEditor is the console's SQL input: monaco (lazy-loaded, schema-aware completion) with a plain textarea as the loading fallback.
export function SqlEditor({
  value,
  onChange,
  readOnly,
  height = "10rem",
  tables = [],
  placeholder,
  onRun,
}: SqlEditorProps) {
  return (
    <div className="overflow-hidden rounded-md border border-border" style={{ height }}>
      <Suspense
        fallback={
          <textarea
            readOnly={readOnly}
            value={value}
            placeholder={placeholder}
            onChange={(e) => onChange?.(e.target.value)}
            onKeyDown={(e) => {
              if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
                e.preventDefault();
                onRun?.();
              }
            }}
            className="size-full resize-none bg-background p-2 font-mono text-xs outline-none"
          />
        }
      >
        <MonacoEditor
          value={value}
          onChange={onChange}
          readOnly={readOnly}
          height={height}
          language="sql"
          tables={tables}
          placeholder={placeholder}
          onRun={onRun}
        />
      </Suspense>
    </div>
  );
}
