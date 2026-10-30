import { ChevronsUpDown, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";

// InstanceFilter picks any number of a driver's connections. The choice shows as chips under the field, because a trigger
// that only says "3 selected" makes the reader open it again to learn which three.
export function InstanceFilter({
  options,
  values,
  onChange,
  label = "Instances",
  emptyLabel = "All instances",
}: {
  options: string[];
  values: string[];
  onChange: (values: string[]) => void;
  label?: string;
  emptyLabel?: string;
}) {
  const toggle = (name: string) =>
    onChange(values.includes(name) ? values.filter((v) => v !== name) : [...values, name]);

  const summary =
    values.length === 0
      ? emptyLabel
      : values.length === 1
        ? values[0]
        : `${values.length} selected`;

  return (
    <div className="flex flex-col gap-1.5">
      <Label>{label}</Label>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={<Button variant="outline" className="w-full justify-between font-normal" />}
        >
          <span className="truncate">{summary}</span>
          <ChevronsUpDown className="size-3.5 opacity-60" />
        </DropdownMenuTrigger>
        <DropdownMenuContent className="max-h-64 overflow-auto">
          {options.length === 0 ? (
            <DropdownMenuItem disabled>No instances registered</DropdownMenuItem>
          ) : (
            options.map((name) => (
              <DropdownMenuCheckboxItem
                key={name}
                checked={values.includes(name)}
                onCheckedChange={() => toggle(name)}
              >
                <span className="truncate">{name}</span>
              </DropdownMenuCheckboxItem>
            ))
          )}
        </DropdownMenuContent>
      </DropdownMenu>
      {values.length > 0 && (
        <div className="flex flex-wrap items-center gap-1">
          {values.map((name) => (
            <Badge key={name} variant="secondary" className="gap-1 pr-1">
              <span className="max-w-32 truncate">{name}</span>
              <button
                type="button"
                aria-label={`remove ${name}`}
                onClick={() => toggle(name)}
                className="rounded-sm p-0.5 hover:bg-background/60"
              >
                <X className="size-3" />
              </button>
            </Badge>
          ))}
          {values.length > 1 && (
            <Button variant="ghost" size="xs" onClick={() => onChange([])}>
              Clear all
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
