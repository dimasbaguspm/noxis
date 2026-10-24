import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";

// InputGroup wraps an input and decorations under one border and focus ring.

export function InputGroup({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="input-group"
      role="group"
      className={cn(
        "group/input-group relative flex w-full min-w-0 items-center rounded-lg border border-input bg-transparent text-base transition-colors outline-none has-[>textarea]:h-auto",
        "focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50",
        "has-aria-invalid:border-destructive has-aria-invalid:ring-3 has-aria-invalid:ring-destructive/20",
        "md:text-sm dark:bg-input/30 dark:has-aria-invalid:border-destructive/50 dark:has-aria-invalid:ring-destructive/40",
        className,
      )}
      {...props}
    />
  );
}

const inputGroupAddonVariants = cva(
  "flex h-auto cursor-text items-center justify-center gap-2 py-1.5 text-sm text-muted-foreground select-none [&>svg:not([class*='size-'])]:size-4",
  {
    variants: {
      align: {
        "inline-start": "order-first pl-2.5",
        "inline-end": "order-last pr-2.5",
        "block-start": "order-first w-full justify-start px-2.5 pt-2",
        "block-end": "order-last w-full justify-start px-2.5 pb-2",
      },
    },
    defaultVariants: { align: "inline-start" },
  },
);

export function InputGroupAddon({
  className,
  align = "inline-start",
  ...props
}: React.ComponentProps<"div"> & VariantProps<typeof inputGroupAddonVariants>) {
  return (
    <div
      role="group"
      data-slot="input-group-addon"
      data-align={align}
      className={cn(inputGroupAddonVariants({ align }), className)}
      onClick={(event) => {
        if ((event.target as HTMLElement).closest("button")) return;
        event.currentTarget.parentElement?.querySelector<HTMLElement>("input, textarea")?.focus();
      }}
      {...props}
    />
  );
}

const inputGroupButtonVariants = cva("flex items-center shadow-none", {
  variants: {
    size: {
      xs: "h-6 gap-1 rounded-sm px-2 text-xs [&_svg:not([class*='size-'])]:size-3.5",
      sm: "h-8 gap-1.5 px-2.5 text-xs [&_svg:not([class*='size-'])]:size-4",
      "icon-xs": "size-6 rounded-sm [&_svg:not([class*='size-'])]:size-3.5",
      "icon-sm": "size-8 [&_svg:not([class*='size-'])]:size-4",
    },
  },
  defaultVariants: { size: "xs" },
});

export function InputGroupButton({
  className,
  type = "button",
  variant = "ghost",
  size = "xs",
  ...props
}: Omit<React.ComponentProps<typeof Button>, "size" | "type"> &
  VariantProps<typeof inputGroupButtonVariants> & {
    type?: "button" | "submit" | "reset";
  }) {
  return (
    <Button
      type={type}
      data-size={size}
      variant={variant}
      className={cn(inputGroupButtonVariants({ size }), className)}
      {...props}
    />
  );
}

export function InputGroupText({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      className={cn(
        "flex items-center gap-2 text-sm text-muted-foreground [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4",
        className,
      )}
      {...props}
    />
  );
}

// The input inside the group is bare: the group owns the border and the ring.
const bare = "flex-1 border-0 bg-transparent shadow-none focus-visible:border-0 focus-visible:ring-0 aria-invalid:ring-0 dark:bg-transparent";

export function InputGroupInput({ className, ...props }: React.ComponentProps<typeof Input>) {
  return <Input data-slot="input-group-control" className={cn(bare, className)} {...props} />;
}

export function InputGroupTextarea({ className, ...props }: React.ComponentProps<typeof Textarea>) {
  return <Textarea data-slot="input-group-control" className={cn(bare, "resize-none", className)} {...props} />;
}
