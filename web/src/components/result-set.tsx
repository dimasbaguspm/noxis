import { DownloadOutlined } from '@ant-design/icons';
import type { ReactNode } from 'react';
import type { Rows } from '../api/types';
import { AppButton } from './app-button';
import { DataTable } from './data-table';
import type { Column } from './data-table';
import { Text } from './text';

// One result set, one rendering: the count the API reported, the export, and the grid. The query and
// the table tab both show rows the engine answered, so neither of them assembles a table itself.

export type ResultRow = Record<string, string | null>;

/** A cell is a value the engine typed as text: null is not empty, so it reads as a dot. */
function gridOf(rows: Rows): { columns: Column<ResultRow>[]; records: ResultRow[] } {
  const columns = (rows.columns ?? []).map((column, index) => ({
    key: `c${index}`,
    title: column.name,
    width: 200,
    render: (row: ResultRow) => (
      <Text mono size="sm">
        {row[`c${index}`] ?? '·'}
      </Text>
    ),
  }));
  const records = (rows.values ?? []).map((values, index) => {
    const record: ResultRow = { key: String(index) };
    values.forEach((cell, column) => {
      record[`c${column}`] = cell;
    });
    return record;
  });
  return { columns, records };
}

export function ResultSet({
  rows,
  name,
  empty = 'the statement returned no rows',
}: {
  rows: Rows;
  /** The file name this result is downloaded as. */
  name: string;
  /** What the body says when the engine answered with no rows. */
  empty?: ReactNode;
}) {
  const { columns, records } = gridOf(rows);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <Text size="xs" tone="muted">
          {rows.row_count} row{rows.row_count === 1 ? '' : 's'}
          {rows.truncated ? " · cut off at the API's row ceiling" : ''}
          {rows.duration_ms ? ` · ${rows.duration_ms} ms` : ''}
        </Text>
        <span className="flex-1" />
        <AppButton
          size="small"
          icon={<DownloadOutlined />}
          disabled={records.length === 0}
          onClick={() => downloadCsv(name, rows.columns, rows.values)}
        >
          Download CSV
        </AppButton>
      </div>
      <DataTable
        rows={records}
        columns={columns}
        rowKey={(row) => String(row.key)}
        compact
        sticky
        minWidth={columns.length * 200}
        empty={
          <Text tone="muted" size="sm">
            {empty}
          </Text>
        }
      />
    </div>
  );
}

// A result is a file somebody will ask for: the rows are already here, so the export is the browser's
// job and not a second request. A null is an empty cell, and a value with a comma, a quote or a
// newline is quoted the way RFC 4180 says, because a spreadsheet is the next reader.

function csvCell(value: string | null): string {
  if (value === null) return '';
  return /[",\r\n]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value;
}

function toCsv(columns: { name: string }[], values: (string | null)[][]): string {
  const header = columns.map((column) => csvCell(column.name)).join(',');
  const lines = values.map((row) => row.map(csvCell).join(','));
  return [header, ...lines].join('\r\n');
}

function downloadCsv(name: string, columns: { name: string }[], values: (string | null)[][]) {
  const blob = new Blob([toCsv(columns, values)], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = name;
  anchor.click();
  // Revoking on the same tick can cancel the download the click just started: the browser reads the
  // blob after the handler returns.
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
