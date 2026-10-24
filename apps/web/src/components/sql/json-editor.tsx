import { Suspense, lazy } from "react";

const MonacoEditor = lazy(() => import("@/components/sql/monaco-editor"));

export interface JsonEditorProps {
  value: string;
  onChange?: (value: string) => void;
  readOnly?: boolean;
  height?: string;
  // the collection's document fields, suggested alongside MongoDB operators
  fields?: string[];
  placeholder?: string;
}

// JsonEditor is the MongoDB input: the same monaco with the document grammar, completing operators and document fields.
export function JsonEditor({
  value,
  onChange,
  readOnly,
  height = "10rem",
  fields = [],
  placeholder,
}: JsonEditorProps) {
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
        <MonacoEditor
          value={value}
          onChange={onChange}
          readOnly={readOnly}
          height={height}
          language="mongo"
          fields={fields}
          placeholder={placeholder}
        />
      </Suspense>
    </div>
  );
}
