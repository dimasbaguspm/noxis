import { Suspense, lazy } from "react";
import type { Table } from "@/lib/api";

const MonacoSqlEditor = lazy(() => import("@/components/sql/monaco-sql-editor"));

export interface SqlEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: string;
  tables?: Table[];
  placeholder?: string;
}

// SqlEditor is the console's SQL input: monaco (lazy-loaded, schema-aware
// completion) with a plain textarea as the loading fallback.
export function SqlEditor({
  value,
  onChange,
  readOnly,
  height = "10rem",
  tables = [],
  placeholder,
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
            className="size-full resize-none bg-background p-2 font-mono text-xs outline-none"
          />
        }
      >
        <MonacoSqlEditor
          value={value}
          onChange={onChange}
          readOnly={readOnly}
          height={height}
          tables={tables}
          placeholder={placeholder}
        />
      </Suspense>
    </div>
  );
}
