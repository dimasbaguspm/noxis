import { clsx } from 'clsx';
import type { ReactNode } from 'react';

// The instance page's facts used to be an Ant Design Description list: evenly sized key/value pairs in
// a grid that Ant Design decides. A panel is the same idea with the direction written down — a title,
// then tiles that wrap when the width runs out — so the page reads as a handful of small blocks rather
// than one wide table of labels.

export function Panel({
  title,
  action,
  children,
  className,
}: {
  title?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={clsx('flex flex-col gap-2.5', className)}>
      {title ? (
        <div className="flex items-center gap-2">
          <h2 className="m-0 text-[11px] font-semibold tracking-[0.08em] text-muted uppercase">
            {title}
          </h2>
          <span className="flex-1" />
          {action}
        </div>
      ) : null}
      {children}
    </section>
  );
}

/** Tiles lays its children out left to right, wrapping at the panel's edge. */
export function Tiles({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={clsx('flex flex-wrap items-start gap-x-8 gap-y-3', className)}>{children}</div>
  );
}

/** Tile is one label and its value: the smallest unit of "what this instance is". */
export function Tile({
  label,
  value,
  mono = false,
  className,
}: {
  label: string;
  value: ReactNode;
  mono?: boolean;
  className?: string;
}) {
  return (
    <div className={clsx('flex min-w-0 flex-col gap-0.5', className)}>
      <span className="text-[11px] tracking-wide text-muted uppercase">{label}</span>
      <span className={clsx('truncate text-sm text-ink', mono && 'font-mono text-[13px]')}>
        {value}
      </span>
    </div>
  );
}
