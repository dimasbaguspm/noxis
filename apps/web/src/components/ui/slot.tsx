import * as React from "react";
import { cn } from "@/lib/utils";

// Slot merges parent props onto one child so asChild keeps the child's native element.
export function Slot({ children, ...props }: React.HTMLAttributes<HTMLElement>) {
  if (!React.isValidElement<Record<string, unknown>>(children)) return null;
  const child = children;
  return React.cloneElement(child, {
    ...props,
    ...child.props,
    className: cn(
      props.className as string | undefined,
      (child.props as { className?: string }).className,
    ),
  });
}

export function SlotChild({
  children,
  asChild = true,
  ...props
}: React.HTMLAttributes<HTMLElement> & { asChild?: boolean }) {
  if (!asChild) return <div {...props}>{children}</div>;
  if (!React.isValidElement<Record<string, unknown>>(children)) return null;
  return React.cloneElement(children, {
    ...props,
    ...children.props,
    className: cn(
      props.className as string | undefined,
      (children.props as { className?: string }).className,
    ),
  });
}
