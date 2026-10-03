import { Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { SqlTab } from "@/hooks/use-sql-tabs";

// SqlTabs is the editor's tab strip: horizontally scrollable, one tab minimum,
// no maximum. The ✕ is hidden for the last remaining tab.
export function SqlTabs({
  tabs,
  activeId,
  onSelect,
  onClose,
  onAdd,
}: {
  tabs: SqlTab[];
  activeId: string;
  onSelect: (id: string) => void;
  onClose: (id: string) => void;
  onAdd: () => void;
}) {
  return (
    <div className="flex items-center gap-1 border-b border-border pb-2">
      <div className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto">
        {tabs.map((tab) => (
          <span
            key={tab.id}
            className={cn(
              "flex shrink-0 items-center gap-1 rounded-md border px-2 py-1 text-xs whitespace-nowrap",
              tab.id === activeId
                ? "border-primary/40 bg-accent text-accent-foreground"
                : "border-border text-muted-foreground hover:bg-accent/50",
            )}
          >
            <button className="max-w-40 truncate" onClick={() => onSelect(tab.id)} title={tab.title}>
              {tab.title}
            </button>
            {tabs.length > 1 && (
              <button
                aria-label={`close ${tab.title}`}
                className="text-muted-foreground hover:text-destructive"
                onClick={() => onClose(tab.id)}
              >
                <X className="size-3" />
              </button>
            )}
          </span>
        ))}
      </div>
      <Button size="icon" variant="ghost" aria-label="new query tab" onClick={onAdd}>
        <Plus />
      </Button>
    </div>
  );
}
