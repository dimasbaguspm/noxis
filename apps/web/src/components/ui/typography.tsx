import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

// Typography is the console's single heading and paragraph scale, so pages do
// not each invent their own font size and colour.

export function H1({ className, ...props }: ComponentProps<"h1">) {
  return <h1 className={cn("text-base font-medium", className)} {...props} />;
}

export function H2({ className, ...props }: ComponentProps<"h2">) {
  return <h2 className={cn("text-sm font-medium", className)} {...props} />;
}

export function P({ className, ...props }: ComponentProps<"p">) {
  return <p className={cn("text-xs text-muted-foreground", className)} {...props} />;
}
