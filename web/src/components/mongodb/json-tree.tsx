import { useState } from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";

// JsonTree renders documents as a collapsible tree: objects and arrays open,
// scalars print, and MongoDB's Extended JSON values stay readable.
export function JsonTree({ value, name }: { value: unknown; name?: string }) {
  if (Array.isArray(value)) {
    return (
      <Branch label={name ?? "array"} count={value.length} bracket="[]">
        {value.map((item, i) => (
          <JsonTree key={`${i}:${JSON.stringify(item)?.slice(0, 24) ?? ""}`} value={item} name={String(i)} />
        ))}
      </Branch>
    );
  }
  if (value && typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>);
    return (
      <Branch label={name ?? "object"} count={entries.length} bracket="{}">
        {entries.map(([key, item]) => (
          <JsonTree key={key} value={item} name={key} />
        ))}
      </Branch>
    );
  }
  return (
    <div className="flex gap-2 py-0.5 pl-4 font-mono text-xs">
      {name !== undefined && <span className="text-muted-foreground">{name}:</span>}
      <span className={scalarClass(value)}>{render(value)}</span>
    </div>
  );
}

function Branch({
  label,
  count,
  bracket,
  children,
}: {
  label: string;
  count: number;
  bracket: string;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(true);
  return (
    <div className="font-mono text-xs">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex items-center gap-1 rounded px-1 py-0.5 hover:bg-accent/50"
      >
        <ChevronRight className={cn("size-3 transition-transform", open && "rotate-90")} />
        <span className="text-muted-foreground">{label}</span>
        <span className="text-muted-foreground/70">
          {bracket[0]}
          {count}
          {bracket[1]}
        </span>
      </button>
      {open && <div className="ml-3 border-l border-border pl-1">{children}</div>}
    </div>
  );
}

function render(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "string") return `"${value}"`;
  return String(value);
}

function scalarClass(value: unknown): string {
  switch (typeof value) {
    case "number":
      return "text-amber-400";
    case "boolean":
      return "text-purple-400";
    case "string":
      return "text-emerald-400";
  }
  return "text-muted-foreground";
}
