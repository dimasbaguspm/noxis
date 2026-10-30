import { Database } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useConnections } from "@/hooks/use-connections";
import type { Capability } from "@/lib/console-nav";

// ConnectionSelect is the page-local instance picker: pages own their choice (the provider only remembers it as the default for the next page).
export function ConnectionSelect({
  capability,
  value,
  onChange,
  className = "w-64",
}: {
  capability: Capability;
  value?: string;
  onChange: (name: string) => void;
  className?: string;
}) {
  const { connections } = useConnections(capability);

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
