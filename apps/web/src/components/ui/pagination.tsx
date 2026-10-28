import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

function pageNumbers(page: number, pages: number): (number | "gap")[] {
  if (pages <= 7) return Array.from({ length: pages }, (_, i) => i + 1);
  const wanted = new Set<number>([1, pages, page - 1, page, page + 1]);
  const nums = [...wanted].filter((n) => n >= 1 && n <= pages).sort((a, b) => a - b);
  const out: (number | "gap")[] = [];
  let prev = 0;
  for (const n of nums) {
    if (n - prev > 1) out.push("gap");
    out.push(n);
    prev = n;
  }
  return out;
}

// Pagination is the shared pager for the list surfaces: numbered buttons plus prev/next, hidden when there is a single page.
export function Pagination({
  page,
  perPage,
  total,
  onChange,
  className,
}: {
  page: number;
  perPage: number;
  total: number;
  onChange: (page: number) => void;
  className?: string;
}) {
  const pages = Math.max(1, Math.ceil(total / perPage));
  if (pages <= 1) return null;
  return (
    <nav
      aria-label="pagination"
      className={cn("flex items-center justify-center gap-1", className)}
    >
      <Button
        size="xs"
        variant="outline"
        aria-label="previous page"
        disabled={page <= 1}
        onClick={() => onChange(page - 1)}
      >
        <ChevronLeft />
      </Button>
      {pageNumbers(page, pages).map((n, i) =>
        n === "gap" ? (
          <span key={`gap-${i}`} className="px-1 text-xs text-muted-foreground">
            …
          </span>
        ) : (
          <Button
            key={n}
            size="xs"
            variant={n === page ? "default" : "outline"}
            onClick={() => onChange(n)}
          >
            {n}
          </Button>
        ),
      )}
      <Button
        size="xs"
        variant="outline"
        aria-label="next page"
        disabled={page >= pages}
        onClick={() => onChange(page + 1)}
      >
        <ChevronRight />
      </Button>
    </nav>
  );
}
