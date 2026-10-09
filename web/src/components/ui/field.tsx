import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

// Field is the reusable form-field set (shadcn/base-field style): a labelled
// input with an optional description and an error slot below it. Every form
// in the console renders validation the same way.

export function FieldSet({ legend, description, children, className }: { legend?: ReactNode; description?: string; children: ReactNode; className?: string }) {
  return (
    <fieldset className={cn("flex flex-col gap-4", className)}>
      {legend && (
        <legend className="grid gap-1">
          {typeof legend === "string" ? <span className="text-sm font-medium">{legend}</span> : legend}
          {description && <span className="text-xs text-muted-foreground">{description}</span>}
        </legend>
      )}
      {children}
    </fieldset>
  );
}

export function FieldGroup({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("flex flex-col gap-3", className)}>{children}</div>;
}

export function Field({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("grid gap-1.5", className)}>{children}</div>;
}

export function FieldLabel({ htmlFor, children, className }: { htmlFor?: string; children: ReactNode; className?: string }) {
  return (
    <label htmlFor={htmlFor} className={cn("text-sm font-medium", className)}>
      {children}
    </label>
  );
}

export function FieldDescription({ children, className }: { children: ReactNode; className?: string }) {
  return <p className={cn("text-xs text-muted-foreground", className)}>{children}</p>;
}

export function FieldError({ children, className }: { children?: ReactNode; className?: string }) {
  if (!children) return null;
  return (
    <p role="alert" className={cn("text-xs text-destructive", className)}>
      {children}
    </p>
  );
}