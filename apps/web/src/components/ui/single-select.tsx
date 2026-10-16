import { cn } from "@/lib/utils";

export interface SelectOption<T extends string = string> {
  value: T;
  label: string;
  detail?: string;
}

// SingleSelect is a radio-backed picker: one option, styled as a full-width
// card, keyboard- and screen-reader-friendly. It stands in for the kind
// choices in the connection wizard (and any one-of list in a form).
export function SingleSelect<T extends string = string>({
  name,
  value,
  onChange,
  options,
  className,
}: {
  name: string;
  value: T | undefined;
  onChange: (value: T) => void;
  options: SelectOption<T>[];
  className?: string;
}) {
  return (
    <div role="radiogroup" aria-label={name} className={cn("grid gap-2", className)}>
      {options.map((option) => {
        const active = value === option.value;
        return (
          <label
            key={option.value}
            className={cn(
              "flex cursor-pointer items-start gap-3 rounded-md border p-3 transition-colors",
              "has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring",
              active
                ? "border-primary/50 bg-accent/50"
                : "border-border hover:bg-accent/30",
            )}
          >
            <input
              type="radio"
              name={name}
              value={option.value}
              checked={active}
              onChange={() => onChange(option.value)}
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