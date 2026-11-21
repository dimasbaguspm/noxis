import { clsx } from 'clsx';
import { useMemo, useState } from 'react';
import { useDatabases, useRows, useTables } from '../api/queries';
import { AsyncState } from '../components/async-state';
import { Card } from '../components/card';
import { EmptyState } from '../components/empty-state';
import { Picker } from '../components/picker';
import { RenderWhen } from '../components/render-when';
import { ResultSet } from '../components/result-set';
import { Text } from '../components/text';

// What is in the tables. MySQL and MariaDB call a schema a database, a target registers one of them,
// and a server usually has more: the registered schema is first because that is the one the instance
// was registered for, and every other schema the server answers stands under it.
//
// The layout is `grid-cols-1` and not a bare `grid`: an implicit column sizes to its content, and a
// result table is wider than a phone, so the page would scroll sideways instead of the table.

export function TablesTab({ ref_, registered }: { ref_: string; registered: string }) {
  const databases = useDatabases(ref_);
  const schemas = useMemo(
    () => registeredFirst(databases.data ?? [], registered),
    [databases.data, registered],
  );

  // The pickers hold what the operator asked for; what is on screen is that, or the first thing the
  // server answered. A schema that disappears between two reads falls back instead of showing nothing.
  const [wantedSchema, setWantedSchema] = useState<string | null>(null);
  const schema =
    wantedSchema && schemas.includes(wantedSchema) ? wantedSchema : (schemas[0] ?? registered);
  const catalog = useTables(ref_, schema);

  const [wantedTable, setWantedTable] = useState<string | null>(null);
  const names = catalog.data?.map((table) => table.name) ?? [];
  const table = wantedTable && names.includes(wantedTable) ? wantedTable : (names[0] ?? null);
  const rows = useRows(ref_, schema, table ?? '');

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(220px,30%)_minmax(0,70%)]">
      <Card
        title="Schema"
        extra={catalog.data ? `${names.length} tables` : undefined}
        bodyClassName="p-3"
      >
        <div className="flex flex-col gap-3">
          <Picker
            label="Schema"
            value={schema || null}
            options={schemas}
            onChange={(next) => {
              setWantedSchema(next);
              setWantedTable(null);
            }}
            loading={databases.isLoading}
            placeholder="no schema"
          />

          <div className="lg:hidden">
            <Picker
              label="Table"
              value={table}
              options={names}
              onChange={setWantedTable}
              loading={catalog.isLoading}
              placeholder="no tables"
            />
          </div>

          <nav aria-label="Tables" className="hidden lg:block">
            <AsyncState
              loading={catalog.isLoading}
              error={catalog.error}
              empty={names.length === 0}
              emptyDescription="The schema has no tables"
              onRetry={() => void catalog.refetch()}
            >
              <ul className="m-0 flex max-h-[26rem] list-none flex-col gap-0.5 overflow-y-auto p-0">
                {catalog.data?.map((entry) => (
                  <li key={entry.name}>
                    <button
                      type="button"
                      onClick={() => setWantedTable(entry.name)}
                      aria-current={entry.name === table ? 'true' : undefined}
                      className={clsx(
                        'flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-sm transition-colors',
                        entry.name === table
                          ? 'bg-brand-soft font-medium text-brand-ink'
                          : 'text-ink hover:bg-surface',
                      )}
                    >
                      <span className="truncate">{entry.name}</span>
                      <span className="flex-1" />
                      <Text size="xs" tone="muted">
                        {compact(entry.rows)}
                      </Text>
                    </button>
                  </li>
                ))}
              </ul>
            </AsyncState>
          </nav>
        </div>
      </Card>

      <Card
        title={table ?? 'Rows'}
        extra={rows.data ? `${rows.data.row_count} rows` : undefined}
        bodyClassName="p-4"
      >
        <RenderWhen
          condition={table}
          fallback={
            <EmptyState
              compact
              description={
                catalog.isLoading ? 'Loading the schema' : 'This schema has no table to read'
              }
            />
          }
        >
          <AsyncState
            loading={rows.isLoading}
            error={rows.error}
            onRetry={() => void rows.refetch()}
          >
            <RenderWhen condition={rows.data}>
              {rows.data ? (
                <ResultSet
                  rows={rows.data}
                  name={`${schema}-${table ?? 'table'}.csv`}
                  empty={`${table ?? 'The table'} has no rows`}
                />
              ) : null}
            </RenderWhen>
          </AsyncState>
        </RenderWhen>
      </Card>
    </div>
  );
}

/** The schema the target was registered with, then the rest as the server answered them. */
function registeredFirst(schemas: string[], registered: string): string[] {
  const seen = new Set<string>();
  const order: string[] = [];
  for (const name of [registered, ...schemas]) {
    if (!name || seen.has(name)) continue;
    seen.add(name);
    order.push(name);
  }
  return order;
}

/** A row estimate is an order of magnitude, not a count: 1.2k reads better than 1284 in a nav. */
function compact(value: number): string {
  if (value < 1000) return String(value);
  if (value < 1_000_000) return `${Math.round(value / 100) / 10}k`;
  return `${Math.round(value / 100_000) / 10}M`;
}
