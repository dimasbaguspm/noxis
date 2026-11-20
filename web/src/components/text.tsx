import { clsx } from 'clsx';
import type { ReactNode } from 'react';

// The words. A heading is a heading at a level, a sentence is a sentence with a tone, a value is
// monospaced or it is not — and no page writes `className="text-xl font-semibold tracking-tight"`
// again. Sizes are the four this console actually uses, not a scale nobody can remember.

type HeadingLevel = 1 | 2 | 3 | 4 | 5 | 6;

const HEADING_SIZE: Record<HeadingLevel, string> = {
  1: 'text-2xl',
  2: 'text-xl',
  3: 'text-lg',
  4: 'text-base',
  5: 'text-sm',
  6: 'text-xs',
};

const HEADING_WEIGHT: Record<HeadingLevel, string> = {
  1: 'font-semibold tracking-tight',
  2: 'font-semibold tracking-tight',
  3: 'font-semibold',
  4: 'font-semibold',
  5: 'font-semibold uppercase tracking-wide text-muted',
  6: 'font-semibold uppercase tracking-wide text-muted',
};

export function Heading({
  level = 2,
  children,
  className,
  id,
}: {
  level?: HeadingLevel;
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  const Tag = `h${level}` as 'h1';
  return (
    <Tag
      id={id}
      className={clsx('m-0 text-ink', HEADING_SIZE[level], HEADING_WEIGHT[level], className)}
    >
      {children}
    </Tag>
  );
}

type Size = 'xs' | 'sm' | 'md' | 'lg';
type Tone = 'default' | 'muted' | 'brand' | 'danger' | 'success';

const TEXT_SIZE: Record<Size, string> = {
  xs: 'text-xs',
  sm: 'text-[13px]',
  md: 'text-sm',
  lg: 'text-base',
};

const TEXT_TONE: Record<Tone, string> = {
  default: 'text-ink',
  muted: 'text-muted',
  brand: 'text-brand',
  danger: 'text-red-600',
  success: 'text-green-700',
};

export function Text({
  children,
  as = 'span',
  size = 'md',
  tone = 'default',
  mono = false,
  className,
}: {
  children: ReactNode;
  as?: 'span' | 'p' | 'div' | 'code' | 'label' | 'em' | 'strong';
  size?: Size;
  tone?: Tone;
  mono?: boolean;
  className?: string;
}) {
  const Tag = as;
  return (
    <Tag className={clsx('m-0', TEXT_SIZE[size], TEXT_TONE[tone], mono && 'font-mono', className)}>
      {children}
    </Tag>
  );
}

/** Eyebrow is the small uppercase label above a block: a section, a tile, a caption. */
export function Eyebrow({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span
      className={clsx(
        'text-[11px] font-semibold tracking-[0.08em] text-muted uppercase',
        className,
      )}
    >
      {children}
    </span>
  );
}
