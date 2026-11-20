import { ApiOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { App as AntApp, Card, Input, Segmented, Tabs, Tag, Typography } from 'antd';
import { useMemo, useState } from 'react';
import { Breadcrumb } from '../components/breadcrumb';
import { Link, useParams } from 'react-router-dom';
import { useProbeTarget, useProposals, useRunQuery, useSchema, useTarget } from '../api/queries';
import { Alert } from '../components/alert';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { DataTable } from '../components/data-table';
import type { Column } from '../components/data-table';
import { Panel, Tile, Tiles } from '../components/panel';
import { RenderWhen } from '../components/render-when';
import { StatusTag } from '../components/sql-block';
import { SqlEditor } from '../editor/sql-editor';
import type { Proposal, Rows, Target } from '../api/types';

// The instance page: what this server is, and the three things an operator does with it. ERD is absent
// on purpose — the API exposes columns and indexes, not foreign keys, and a diagram from half a graph
// is a confident lie.

type Tab = 'query' | 'dml' | 'ddl';
type Row = Record<string, string | null>;

export function TargetPage() {
  const { ref = '' } = useParams();
  const [tab, setTab] = useState<Tab>('query');
  const target = useTarget(ref);

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
                    <h1 id="target-title" className="m-0 text-xl font-semibold">
                      {target.data.name}
                    </h1>
                    <Tag>{target.data.driver}</Tag>
                    <RenderWhen condition={target.data.server_version}>
                      <Tag>{target.data.server_version}</Tag>
                    </RenderWhen>
                    <RenderWhen condition={target.data.archived_at}>
                      <Tag>archived</Tag>
                    </RenderWhen>
                  </div>

                  <Panel>
                    <Tiles>
                      <Tile
                        label="Host"
                        mono
                        value={`${target.data.username}@${target.data.host}:${target.data.port}`}
                      />
                      <Tile label="Database" mono value={target.data.database} />
                      <Tile label="TLS" value={target.data.tls_mode} />
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
                  { key: 'dml', label: 'DML', children: <Queue target={ref} kind="dml" /> },
                  { key: 'ddl', label: 'DDL', children: <Queue target={ref} kind="ddl" /> },
                ]}
              />
            </>
          ) : null}
        </RenderWhen>
      </AsyncState>
    </section>
  );
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
        <span className="text-[11px] font-semibold tracking-[0.08em] text-muted uppercase">
          Connectivity
        </span>
        <RenderWhen condition={target.last_probe_status}>
          <Tag color={ok ? 'green' : 'red'} className="m-0">
            {ok ? 'reachable' : 'unreachable'}
          </Tag>
        </RenderWhen>
        <RenderWhen condition={!target.last_probe_status}>
          <Tag className="m-0">never probed</Tag>
        </RenderWhen>
        <span className="text-xs text-muted">
          {target.last_probe_at
            ? `answered ${new Date(target.last_probe_at).toLocaleString()}`
            : 'not contacted yet'}
          {target.server_version ? ` · ${target.server_version}` : ''}
        </span>
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
  const run = useRunQuery(ref_);
  const schema = useSchema(ref_, database);

  const columns: Column<Row>[] = useMemo(
    () =>
      (run.data?.columns ?? []).map((column, index) => ({ key: `c${index}`, title: column.name })),
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
      <Alert tone="info" title="SELECT, EXPLAIN and SHOW only" />
      <SqlEditor value={sql} onChange={setSql} schema={schema.data ?? null} height={200} />
      <div className="flex gap-2">
        <AppButton
          intent="primary"
          icon={<PlayCircleOutlined />}
          loading={run.isPending}
          onClick={() => run.mutate({ sql, limit: 200 })}
        >
          Run
        </AppButton>
      </div>
      <RenderWhen condition={run.error}>
        <Alert
          tone="error"
          title="That statement did not run"
          description={run.error instanceof Error ? run.error.message : 'failed'}
        />
      </RenderWhen>
      <RenderWhen condition={run.data}>
        <Result rows={run.data as Rows} columns={columns} records={records} />
      </RenderWhen>
    </div>
  );
}

function Result({
  rows,
  columns,
  records,
}: {
  rows: Rows;
  columns: Column<Row>[];
  records: Row[];
}) {
  return (
    <div className="flex flex-col gap-2">
      <Typography.Text type="secondary" className="text-xs">
        {rows.row_count} row{rows.row_count === 1 ? '' : 's'}
        {rows.truncated ? ' · truncated by the limit the API applied' : ''}
      </Typography.Text>
      <DataTable rows={records} columns={columns} rowKey={(row) => String(row.key)} compact />
    </div>
  );
}

function Queue({ target, kind }: { target: string; kind?: 'dml' | 'ddl' }) {
  const [status, setStatus] = useState('all');
  const [term, setTerm] = useState('');
  const proposals = useProposals({
    target,
    kind,
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

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          allowClear
          placeholder="Filter by title"
          value={term}
          onChange={(event) => setTerm(event.target.value)}
          aria-label="Filter proposals by title"
          className="max-w-[280px]"
        />
        <Segmented
          value={status}
          onChange={(value) => setStatus(String(value))}
          options={['all', 'verified', 'approved', 'applied', 'failed']}
        />
        <span className="flex-1" />
        <Typography.Text type="secondary" className="text-xs">
          {visible.length} of {items.length}
        </Typography.Text>
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
          columns={[
            {
              key: 'title',
              title: 'Title',
              render: (row) => (
                <Link to={`/proposals/${row.ref}`} className="text-brand hover:underline">
                  {row.title}
                </Link>
              ),
            },
            {
              key: 'status',
              title: 'Status',
              render: (row) => <StatusTag status={row.status} />,
            },
            { key: 'proposed_by', title: 'Proposed by', showFrom: 'md' },
          ]}
        />
      </AsyncState>
    </div>
  );
}
