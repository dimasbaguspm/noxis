import { cn } from "@/lib/utils";
import type { SelectOption } from "@/components/ui/single-select";

// MultiSelect is a checkbox-backed picker: any number of options, one card
// per row. Used for multi-value filters and (later) batch group grants.
export function MultiSelect<T extends string = string>({
  name,
  values,
  onChange,
  options,
  className,
}: {
  name: string;
  values: T[];
  onChange: (values: T[]) => void;
  options: SelectOption<T>[];
  className?: string;
}) {
  function toggle(value: T) {
    onChange(values.includes(value) ? values.filter((v) => v !== value) : [...values, value]);
  }

  return (
    <div role="group" aria-label={name} className={cn("grid gap-2", className)}>
      {options.map((option) => {
        const checked = values.includes(option.value);
        return (
          <label
            key={option.value}
            className={cn(
              "flex cursor-pointer items-start gap-3 rounded-md border p-3 transition-colors",
              "has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring",
              checked ? "border-primary/50 bg-accent/50" : "border-border hover:bg-accent/30",
            )}
          >
            <input
              type="checkbox"
              name={name}
              value={option.value}
              checked={checked}
              onChange={() => toggle(option.value)}
              className="mt-0.5 size-4 accent-foreground"
            />
            <span className="grid gap-0.5">
              <span className="text-sm font-medium">{option.label}</span>
              {option.detail && <span className="text-xs text-muted-foreground">{option.detail}</span>}
            </span>
          </label>
        );
      })}
    </div>
  );
}