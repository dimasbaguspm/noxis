import { Database } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Connection } from "@/lib/api";

// ConnectionSelect is the page-local instance picker: the page owns the choice and the list it offers, so a picker can
// never show an instance the page is not allowed to act on.
export function ConnectionSelect({
  connections,
  value,
  onChange,
  className = "w-64",
}: {
  connections: Connection[];
  value?: string;
  onChange: (name: string) => void;
  className?: string;
}) {
  if (connections.length === 0) {
    return (
      <span className="text-xs text-muted-foreground">
        No connections yet — add one from the dashboard
      </span>
    );
  }

  return (
    <Select value={value} onValueChange={(next) => onChange(String(next))}>
      <SelectTrigger className={className} aria-label="DB instance">
        <Database className="size-3.5 text-muted-foreground" />
        <SelectValue placeholder="Pick an instance" />
      </SelectTrigger>
      <SelectContent>
        {connections.map((c) => (
          <SelectItem key={c.name} value={c.name ?? ""}>
            <Database className="size-3.5 text-muted-foreground" />
            {c.name} · {c.driver}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
