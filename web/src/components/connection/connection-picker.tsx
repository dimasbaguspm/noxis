import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useFamily } from "@/app/app-provider";

// ConnectionPicker switches the active connection — the one shared piece of
// context every page reads.
export function ConnectionPicker() {
  const { connections, active, setActive } = useFamily("sql");

  if (connections.length === 0) {
    return <span className="text-xs text-muted-foreground">no connections yet</span>;
  }

  return (
    <Select value={active} onValueChange={(value) => setActive(String(value))}>
      <SelectTrigger className="w-56" aria-label="active connection">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {connections.map((c) => (
          <SelectItem key={c.name} value={c.name}>
            {c.name} · {c.driver}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
