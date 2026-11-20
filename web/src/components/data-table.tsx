import { Table } from 'antd';
import type { Breakpoint } from 'antd';
import type { ReactNode } from 'react';

// One table. Ant Design's is a good engine and a fiddly API: `dataIndex`, `render(_, record)`, a
// `responsive` array of breakpoints, a `rowKey` that is sometimes a function. A screen names its
// columns as `{ key, title, render(row) }` and never touches the rest, so the five tables that will
// exist share one shape and one set of styles (`.noxis-data-table` in styles.css).

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
};

const BREAKPOINTS: Breakpoint[] = ['xs', 'sm', 'md', 'lg', 'xl', 'xxl'];

export function DataTable<T>({
  rows,
  columns,
  rowKey,
  empty,
  loading = false,
  compact = false,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T) => string;
  /** What the body says when there are no rows. */
  empty?: ReactNode;
  loading?: boolean;
  compact?: boolean;
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
        scroll={{ x: 'max-content' }}
        columns={columns.map((column) => ({
          key: column.key,
          dataIndex: column.key,
          title: column.title,
          align: column.align,
          width: column.width,
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
