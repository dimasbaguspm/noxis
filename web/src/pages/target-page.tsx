import {
  ApiOutlined,
  DeleteOutlined,
  DownloadOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import { App as AntApp, Input, Segmented, Tabs, Tag } from 'antd';
import { useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Can } from '../api/permissions';
import { Breadcrumb } from '../components/breadcrumb';
import { Card } from '../components/card';
import { Link, useParams } from 'react-router-dom';
import {
  useArchiveTarget,
  useProbeTarget,
  useProposals,
  useRunQuery,
  useSchema,
  useTables,
  useTarget,
} from '../api/queries';
import { Alert } from '../components/alert';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { CreateProposal } from '../components/create-proposal';
import { DataTable } from '../components/data-table';
import { EngineMark } from '../components/engine-mark';
import type { Column } from '../components/data-table';
import { Panel, Tile, Tiles } from '../components/panel';
import { RenderWhen } from '../components/render-when';
import { StatusTag } from '../components/sql-block';
import { Statistic } from '../components/statistic';
import { Eyebrow, Heading, Text } from '../components/text';
import { SqlEditor } from '../editor/sql-editor';
import type { Schema } from '../editor/sql-editor';
import type { Proposal, Rows, Target } from '../api/types';

// The instance page: what this server is, and the three things an operator does with it. ERD is absent
// on purpose — the API exposes columns and indexes, not foreign keys, and a diagram from half a graph
// is a confident lie.

type Tab = 'query' | 'changes';
type Row = Record<string, string | null>;

export function TargetPage() {
  const { ref = '' } = useParams();
  const [tab, setTab] = useState<Tab>('query');
  const target = useTarget(ref);
  const tables = useTables(ref, target.data?.database ?? '');
  const { modal, message } = AntApp.useApp();
  const navigate = useNavigate();
  const archive = useArchiveTarget();

  // Archiving is this product's delete: nothing is removed and the journal keeps it. The confirmation
  // says so instead of implying a second click brings it back.
  const archiveIt = () => {
    modal.confirm({
      title: `Archive ${target.data?.name ?? 'this instance'}?`,
      content:
        'It leaves the list. Its proposals and journal stay, and the name can be registered again. There is no un-archive yet.',
      okText: 'Archive',
      okButtonProps: { danger: true },
      onOk: async () => {
        await archive.mutateAsync(ref);
        void message.success('archived');
        navigate('/');
      },
    });
  };

  return (
    <section className="flex flex-col gap-5" aria-labelledby="target-title">
      <Breadcrumb
        items={[{ label: 'Instances', to: '/' }, { label: target.data?.name ?? 'Instance' }]}
      />

      <AsyncState
        loading={target.isLoading}
        error={target.error}
        onRetry={() => void target.refetch()}
      >
        <RenderWhen condition={target.data}>
          {target.data ? (
            <>
              <Card>
                <div className="flex flex-col gap-4">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <Heading level={1} id="target-title">
                      {target.data.name}
                    </Heading>
                    <EngineMark engine={target.data.driver} size={16} />
                    <RenderWhen condition={target.data.archived_at}>
                      <Tag>archived</Tag>
                    </RenderWhen>
                    <span className="flex-1" />
                    <Can do="delete:target">
                      <RenderWhen condition={!target.data.archived_at}>
                        <AppButton intent="quiet" icon={<DeleteOutlined />} onClick={archiveIt}>
                          Archive
                        </AppButton>
                      </RenderWhen>
                    </Can>
                  </div>

                  <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                    <Statistic label="Version" value={target.data.server_version ?? '—'} />
                    <Statistic
                      label="Tables"
                      value={tables.data ? tables.data.length : '—'}
                      hint={target.data.database}
                    />
                    <Statistic label="Port" value={target.data.port} hint={target.data.tls_mode} />
                    <Statistic
                      label="Last answer"
                      value={
                        target.data.last_probe_at
                          ? new Date(target.data.last_probe_at).toLocaleTimeString([], {
                              hour: '2-digit',
                              minute: '2-digit',
                            })
                          : '—'
                      }
                      hint={target.data.last_probe_status ?? 'never probed'}
                    />
                  </div>

                  <Panel>
                    <Tiles>
                      <Tile
                        label="Host"
                        mono
                        value={`${target.data.username}@${target.data.host}`}
                      />
                      <Tile label="Database" mono value={target.data.database} />
                      <Tile label="Registered by" value={target.data.created_by} />
                    </Tiles>
                  </Panel>

                  <Connectivity target={target.data} />
                </div>
              </Card>

              <Tabs
                activeKey={tab}
                onChange={(key) => setTab(key as Tab)}
                items={[
                  {
                    key: 'query',
                    label: 'Query',
                    children: <QueryTab ref_={ref} database={target.data.database} />,
                  },
                  {
                    key: 'changes',
                    label: 'Changes',
                    children: (
                      <Changes target={ref} schema={schemaOf(target.data.database, tables.data)} />
                    ),
                  },
                ]}
              />
            </>
          ) : null}
        </RenderWhen>
      </AsyncState>
    </section>
  );
}

/** The completion list a proposal form gets: table names, without the column detail the editor loads. */
function schemaOf(database: string, tables: { name: string }[] | undefined): Schema | null {
  if (!tables) return null;
  return { database, tables: tables.map((table) => ({ name: table.name, columns: [] })) };
}

// Connectivity is why an operator opens this page: one button, the engine's own answer, and the time
// it arrived. A failed probe is stored on the target, so a refresh does not forget it.
function Connectivity({ target }: { target: Target }) {
  const { message } = AntApp.useApp();
  const probe = useProbeTarget();
  const [failure, setFailure] = useState<string | null>(null);
  const ok = target.last_probe_status === 'ok';

  return (
    <div className="flex flex-col gap-2 border-t border-line pt-3">
      <div className="flex flex-wrap items-center gap-2.5">
        <Eyebrow>Connectivity</Eyebrow>
        <RenderWhen condition={target.last_probe_status}>
          <Tag color={ok ? 'green' : 'red'} className="m-0">
            {ok ? 'reachable' : 'unreachable'}
          </Tag>
        </RenderWhen>
        <RenderWhen condition={!target.last_probe_status}>
          <Tag className="m-0">never probed</Tag>
        </RenderWhen>
        <Text size="xs" tone="muted">
          {target.last_probe_at
            ? `answered ${new Date(target.last_probe_at).toLocaleString()}`
            : 'not contacted yet'}
        </Text>
        <span className="flex-1" />
        <AppButton
          icon={<ApiOutlined />}
          loading={probe.isPending}
          onClick={() => {
            setFailure(null);
            probe.mutate(target.ref, {
              onSuccess: (data) => {
                void message.success({
                  key: 'probe',
                  content: `Answered as ${data.capabilities.version || data.capabilities.driver}`,
                });
              },
              onError: (error) =>
                setFailure(error instanceof Error ? error.message : String(error)),
            });
          }}
        >
          Test connection
        </AppButton>
      </div>
      <RenderWhen condition={failure}>
        <Alert tone="error" title="The instance did not answer" description={failure} />
      </RenderWhen>
    </div>
  );
}

function QueryTab({ ref_, database }: { ref_: string; database: string }) {
  const [sql, setSql] = useState('SELECT 1 AS one;');
  const [took, setTook] = useState<number | null>(null);
  const started = useRef(0);
  const run = useRunQuery(ref_);
  const schema = useSchema(ref_, database);

  const kick = () => {
    started.current = performance.now();
    run.mutate({ sql }, { onSuccess: () => setTook(performance.now() - started.current) });
  };

  const columns: Column<Row>[] = useMemo(
    () =>
      (run.data?.columns ?? []).map((column, index) => ({
        key: `c${index}`,
        title: column.name,
        width: 200,
        render: (row) => (
          <Text mono size="sm">
            {row[`c${index}`] ?? '·'}
          </Text>
        ),
      })),
    [run.data],
  );

  const records: Row[] = useMemo(() => {
    const values: (string | null)[][] = run.data?.values ?? [];
    return values.map((row, index) => {
      const record: Row = { key: String(index) };
      row.forEach((cell, column) => {
        record[`c${column}`] = cell;
      });
      return record;
    });
  }, [run.data]);

  return (
    <div className="flex flex-col gap-3">
      <Card
        title="Statement"
        extra={
          schema.data
            ? `${schema.data.tables.length} tables · read only`
            : schema.isLoading
              ? 'loading the schema'
              : 'read only'
        }
        footer={
          <>
            <AppButton
              intent="primary"
              icon={<PlayCircleOutlined />}
              loading={run.isPending}
              onClick={kick}
            >
              Run
            </AppButton>
            <Text size="xs" tone="muted">
              ⌘ / Ctrl + ↵ · SELECT, EXPLAIN and SHOW only
            </Text>
          </>
        }
      >
        <SqlEditor
          value={sql}
          onChange={setSql}
          onRun={kick}
          schema={schema.data ?? null}
          height={220}
        />
      </Card>

      <RenderWhen condition={run.error}>
        <Alert
          tone="error"
          title="That statement did not run"
          description={run.error instanceof Error ? run.error.message : 'failed'}
        />
      </RenderWhen>
      <RenderWhen condition={run.data}>
        <Result
          rows={run.data as Rows}
          columns={columns}
          records={records}
          took={took}
          name={`${database}-${new Date().toISOString().slice(0, 19).replaceAll(':', '')}.csv`}
        />
      </RenderWhen>
    </div>
  );
}
function Result({
  rows,
  columns,
  records,
  took,
  name,
}: {
  rows: Rows;
  columns: Column<Row>[];
  records: Row[];
  took: number | null;
  /** The file name this result is downloaded as. */
  name: string;
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <Text size="xs" tone="muted">
          {rows.row_count} row{rows.row_count === 1 ? '' : 's'}
          {rows.truncated ? " · cut off at the API's row ceiling" : ''}
          {took ? ` · ${Math.round(took)} ms` : ''}
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
            the statement returned no rows
          </Text>
        }
      />
    </div>
  );
}

function Changes({ target, schema }: { target: string; schema: Schema | null }) {
  const navigate = useNavigate();
  const [status, setStatus] = useState('all');
  const [term, setTerm] = useState('');
  const proposals = useProposals({
    target,
    status: status === 'all' ? undefined : status,
    limit: 50,
  });
  const items = proposals.data?.items ?? [];
  const needle = term.trim().toLowerCase();
  const visible = needle
    ? items.filter((item) =>
        [item.title, item.query, item.proposed_by].some((field) =>
          field.toLowerCase().includes(needle),
        ),
      )
    : items;

  const open = (ref: string) => navigate(`/targets/${target}/proposals/${ref}`);

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          allowClear
          placeholder="Filter by title"
          value={term}
          onChange={(event) => setTerm(event.target.value)}
          aria-label="Filter changes by title"
          className="max-w-[240px]"
        />
        <div className="max-w-full overflow-x-auto">
          <Segmented
            value={status}
            onChange={(value) => setStatus(String(value))}
            options={['all', 'verified', 'approved', 'applied', 'failed']}
          />
        </div>
        <span className="flex-1" />
        <Text size="xs" tone="muted">
          {visible.length} of {items.length}
        </Text>
        <CreateProposal target={target} schema={schema} />
      </div>

      <AsyncState
        loading={proposals.isLoading}
        error={proposals.error}
        empty={visible.length === 0}
        emptyDescription="Nothing matches"
      >
        <DataTable<Proposal>
          rows={visible}
          rowKey={(row) => row.ref}
          compact
          onRow={(row) => ({
            onClick: () => open(row.ref),
            onKeyDown: (event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                open(row.ref);
              }
            },
            tabIndex: 0,
            role: 'link',
            'aria-label': `Open ${row.title}`,
            className: 'cursor-pointer',
          })}
          columns={[
            {
              key: 'title',
              title: 'Title',
              render: (row) => (
                <Link
                  to={`/targets/${target}/proposals/${row.ref}`}
                  className="text-brand hover:underline"
                >
                  {row.title}
                </Link>
              ),
            },
            {
              key: 'status',
              title: 'Status',
              width: 130,
              render: (row) => <StatusTag status={row.status} />,
            },
            { key: 'proposed_by', title: 'Proposed by', showFrom: 'md' },
          ]}
        />
      </AsyncState>
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
