import { Table } from 'antd';
import type { Breakpoint } from 'antd';
import type { ReactNode } from 'react';

// One table. Ant Design's is a good engine and a fiddly API: `dataIndex`, `render(_, record)`, a
// `responsive` array of breakpoints, a `rowKey` that is sometimes a function. A screen names its
// columns as `{ key, title, render(row) }` and never touches the rest, so every table in the console
// shares one shape and one set of styles (`.noxis-data-table` in styles.css).
//
// A wide table scrolls and keeps its first column: `fixed: 'left'` plus a numeric `minWidth` is the
// Ant Design recipe, named once here so a page does not have to know it.

export type Column<T> = {
  /** The field the column reads, and the column's identity. */
  key: string;
  title: string;
  /** What a cell shows; the default is the field itself. */
  render?: (row: T) => ReactNode;
  align?: 'left' | 'center' | 'right';
  width?: number | string;
  /** The breakpoint this column appears at; below it, the column is gone rather than squeezed. */
  showFrom?: Breakpoint;
  /** Pins the column while the rest scrolls. Needs a width to have something to pin. */
  fixed?: 'left' | 'right';
  /** Values the header can be narrowed to. The filter matches the row's own field, named by `key`. */
  filters?: { text: string; value: string }[];
  /** For a column whose filter is not a straight comparison — a missing value read as "never". */
  onFilter?: (value: string, row: T) => boolean;
};

const BREAKPOINTS: Breakpoint[] = ['xs', 'sm', 'md', 'lg', 'xl', 'xxl'];

export function DataTable<T>({
  rows,
  columns,
  rowKey,
  empty,
  loading = false,
  compact = false,
  minWidth,
  sticky = false,
  onRow,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  /** What the body says when there are no rows. */
  empty?: ReactNode;
  loading?: boolean;
  compact?: boolean;
  /** The width the table keeps before it scrolls sideways; a fixed column needs a number here. */
  minWidth?: number;
  /** Keeps the header in view while the body scrolls. */
  sticky?: boolean;
  /** What a whole row does when it is clicked or activated from the keyboard. */
  onRow?: (row: T) => React.HTMLAttributes<HTMLElement>;
}) {
  return (
    <div className="noxis-data-table">
      <Table<T>
        size={compact ? 'small' : 'middle'}
        rowKey={rowKey}
        loading={loading}
        pagination={false}
        dataSource={rows}
        locale={empty ? { emptyText: empty } : undefined}
        sticky={sticky}
        scroll={minWidth ? { x: minWidth } : { x: 'max-content' }}
        onRow={onRow ? (row) => onRow(row) : undefined}
        columns={columns.map((column) => ({
          key: column.key,
          dataIndex: column.key,
          title: column.title,
          align: column.align,
          width: column.width,
          fixed: column.fixed,
          filters: column.filters,
          // A filtered column has to be able to answer for itself: the value Ant Design passes back
          // is compared with the row's own field, so a screen names the column once and never a
          // second `onFilter` that repeats what `key` already says.
          onFilter: column.onFilter
            ? (value, row) => column.onFilter!(String(value), row)
            : column.filters
              ? (value, row) =>
                  String((row as Record<string, unknown>)[column.key]) === String(value)
              : undefined,
          responsive: column.showFrom
            ? BREAKPOINTS.slice(BREAKPOINTS.indexOf(column.showFrom))
            : undefined,
          render: (value: unknown, row: T) =>
            column.render ? column.render(row) : (value as ReactNode),
        }))}
      />
    </div>
  );
}
