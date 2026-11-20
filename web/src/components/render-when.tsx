import type { ReactNode } from 'react';

// RenderWhen is the one conditional a screen should need: it names the condition, so a ternary in the
// middle of JSX is not the only option and a `&&` that renders `0` is impossible.
export function RenderWhen({
  condition,
  children,
  fallback = null,
}: {
  condition: unknown;
  children: ReactNode;
  fallback?: ReactNode;
}) {
  return <>{condition ? children : fallback}</>;
}
