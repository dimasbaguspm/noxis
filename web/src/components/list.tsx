import { clsx } from 'clsx';
import type { ReactNode } from 'react';

// A list of sentences is not an Ant Design `List`: that one is a data list with rows, actions and
// pagination, and it renders a `<ul>` with none of the semantics a reader of prose expects. What is
// needed here is the list a document has — ordered or not, one item per line, a marker in the brand
// colour — so it is the element, wrapped once.

export function List({
  children,
  ordered = false,
  className,
}: {
  children: ReactNode;
  ordered?: boolean;
  className?: string;
}) {
  const Tag = ordered ? 'ol' : 'ul';
  return (
    <Tag
      className={clsx(
        'm-0 flex list-inside flex-col gap-1.5 pl-1 text-sm leading-relaxed text-muted',
        ordered ? 'list-decimal' : 'list-disc',
        'marker:text-brand',
        className,
      )}
    >
      {children}
    </Tag>
  );
}

export function ListItem({ children }: { children: ReactNode }) {
  return <li className="pl-1">{children}</li>;
}
