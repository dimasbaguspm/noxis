import { PlayCircleOutlined } from '@ant-design/icons';
import { Alert, Card, Descriptions, Segmented, Table, Tabs, Tag, Typography, Input } from 'antd';
import { useMemo, useState } from 'react';
import { Breadcrumb } from '../components/breadcrumb';
import { Link, useParams } from 'react-router-dom';
import { useProposals, useRunQuery, useSchema, useTarget } from '../api/queries';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { RenderWhen } from '../components/render-when';
import { StatusTag } from '../components/sql-block';
import { SqlEditor } from '../editor/sql-editor';
import type { Proposal, Rows } from '../api/types';

// The instance page: what this server is, and the four things an operator does with it. ERD is absent
// on purpose — the API exposes columns and indexes, not foreign keys, and a diagram from half a graph
// is a confident lie.

type Tab = 'query' | 'dml' | 'ddl';

export function TargetPage() {
  const { ref = '' } = useParams();
  const [tab, setTab] = useState<Tab>('query');
  const target = useTarget(ref);

  return (
    <section className="flex flex-col gap-5" aria-labelledby="target-title">
      <Breadcrumb items={[{ label: 'Instances', to: '/' }, { label: target.data?.name ?? 'Instance' }]} />

      <AsyncState loading={target.isLoading} error={target.error} onRetry={() => void target.refetch()}>
        <RenderWhen condition={target.data}>
          {target.data ? (
            <>
              <Card>
                <div className="flex flex-col gap-3">
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
                  <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 3 }}>
                    <Descriptions.Item label="Host">
                      <span className="font-mono text-xs">
                        {target.data.username}@{target.data.host}:{target.data.port}
                      </span>
                    </Descriptions.Item>
                    <Descriptions.Item label="Database">{target.data.database}</Descriptions.Item>
                    <Descriptions.Item label="TLS">{target.data.tls_mode}</Descriptions.Item>
                    <Descriptions.Item label="Password">
                      {target.data.has_password ? 'sealed' : 'none'}
                    </Descriptions.Item>
                    <Descriptions.Item label="Probe">
                      {target.data.last_probe_status ?? 'never'}
                    </Descriptions.Item>
                    <Descriptions.Item label="Registered by">
                      {target.data.created_by}
                    </Descriptions.Item>
                  </Descriptions>
                </div>
              </Card>

              <Tabs
                activeKey={tab}
                onChange={(key) => setTab(key as Tab)}
                items={[
                  { key: 'query', label: 'Query', children: <QueryTab ref_={ref} database={target.data.database} /> },
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

function QueryTab({ ref_, database }: { ref_: string; database: string }) {
  const [sql, setSql] = useState('SELECT 1 AS one;');
  const run = useRunQuery(ref_);
  const schema = useSchema(ref_, database);

  const columns = useMemo(
    () =>
      (run.data?.columns ?? []).map((column, index) => ({
        title: column.name,
        dataIndex: `c${index}`,
        key: column.name,
      })),
    [run.data],
  );

  const rows = useMemo(() => {
    const values: (string | null)[][] = run.data?.values ?? [];
    return values.map((row, index) => {
      const record: Record<string, string | null> = { key: String(index) };
      row.forEach((cell, column) => {
        record[`c${column}`] = cell;
      });
      return record;
    });
  }, [run.data]);

  return (
    <div className="flex flex-col gap-3">
      <Alert type="info" showIcon message="SELECT, EXPLAIN and SHOW only" />
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
        <Alert type="error" showIcon message={run.error instanceof Error ? run.error.message : 'failed'} />
      </RenderWhen>
      <RenderWhen condition={run.data}>
        <Result rows={run.data as Rows} columns={columns} records={rows} />
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
  columns: { title: string; dataIndex: string; key: string }[];
  records: Record<string, string | null>[];
}) {
  return (
    <div className="flex flex-col gap-2">
      <Typography.Text type="secondary" className="text-xs">
        {rows.row_count} row{rows.row_count === 1 ? '' : 's'}
        {rows.truncated ? ' · truncated by the limit the API applied' : ''}
      </Typography.Text>
      <Table
        size="small"
        rowKey="key"
        pagination={false}
        scroll={{ x: 'max-content' }}
        dataSource={records}
        columns={columns}
      />
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
        <Table<Proposal>
          size="small"
          rowKey="ref"
          pagination={false}
          scroll={{ x: 'max-content' }}
          dataSource={visible}
          columns={[
            {
              title: 'Title',
              dataIndex: 'title',
              render: (_: unknown, row) => (
                <Link to={`/proposals/${row.ref}`} className="text-brand hover:underline">
                  {row.title}
                </Link>
              ),
            },
            { title: 'Status', dataIndex: 'status', render: (value: string) => <StatusTag status={value} /> },
            { title: 'By', dataIndex: 'proposed_by', responsive: ['md'] },
          ]}
        />
      </AsyncState>
    </div>
  );
}
