import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  App as AntApp,
  Button,
  Card,
  Descriptions,
  Empty,
  Flex,
  Input,
  Segmented,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { Link, useParams } from 'react-router-dom';
import { useSession } from '../api/session';
import { Async, useLoader } from '../components/Async';
import { SqlEditor } from '../editor/SqlEditor';
import type { Schema } from '../editor/SqlEditor';
import type { Proposal, Rows, Target, TableDetail } from '../api/types';

// The instance page: what this server is, and the four things an operator does with it. Query runs a
// statement; DML and DDL are the proposal queue for that kind; History is everything proposed against
// this target. ERD is deliberately absent: the API knows columns and indexes, not foreign keys, and a
// diagram drawn from half the graph is a confident lie.

type Tab = 'query' | 'dml' | 'ddl' | 'history';

export function TargetPage() {
  const { ref = '' } = useParams();
  const { api } = useSession();
  const [tab, setTab] = useState<Tab>('query');
  const [target, loadTarget] = useLoader<Target>();

  useEffect(() => {
    if (!api) return;
    void loadTarget(() => api.target(ref));
  }, [api, ref, loadTarget]);

  return (
    <Flex vertical gap={20}>
      <Flex align="center" gap={8} wrap>
        <Link to="/">← Instances</Link>
      </Flex>

      <Async state={target} retry={() => api && void loadTarget(() => api.target(ref))}>
        {(found) => (
          <>
            <Card>
              <Flex vertical gap={12}>
                <Flex align="center" gap={10} wrap>
                  <Typography.Title level={3} style={{ margin: 0 }}>
                    {found.name}
                  </Typography.Title>
                  <Tag>{found.driver}</Tag>
                  {found.server_version ? <Tag>{found.server_version}</Tag> : null}
                  {found.archived_at ? <Tag>archived</Tag> : null}
                </Flex>
                <Descriptions size="small" column={{ xs: 1, sm: 2, lg: 3 }}>
                  <Descriptions.Item label="Host">
                    <span className="noxis-mono">
                      {found.username}@{found.host}:{found.port}
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="Database">{found.database}</Descriptions.Item>
                  <Descriptions.Item label="TLS">{found.tls_mode}</Descriptions.Item>
                  <Descriptions.Item label="Password">
                    {found.has_password ? 'sealed' : 'none'}
                  </Descriptions.Item>
                  <Descriptions.Item label="Probe">
                    {found.last_probe_status ?? 'never'}
                  </Descriptions.Item>
                  <Descriptions.Item label="Registered by">{found.created_by}</Descriptions.Item>
                </Descriptions>
              </Flex>
            </Card>

            <Tabs
              activeKey={tab}
              onChange={(key) => setTab(key as Tab)}
              items={[
                { key: 'query', label: 'Query', children: <QueryTab target={found} /> },
                { key: 'dml', label: 'DML', children: <QueueTab target={found} kind="dml" /> },
                { key: 'ddl', label: 'DDL', children: <QueueTab target={found} kind="ddl" /> },
                { key: 'history', label: 'History', children: <QueueTab target={found} /> },
              ]}
            />
          </>
        )}
      </Async>
    </Flex>
  );
}

function QueryTab({ target }: { target: Target }) {
  const { api } = useSession();
  const { message } = AntApp.useApp();
  const [sql, setSql] = useState('SELECT 1 AS one;');
  const [rows, setRows] = useState<Rows | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [schema, setSchema] = useState<Schema | null>(null);

  // The schema is what the editor completes from: one read of the tables, then one per table for its
  // columns. It is the target's own default database, because that is what a statement without a
  // qualifier runs against.
  useEffect(() => {
    if (!api) return;
    let live = true;
    void (async () => {
      try {
        const tables = await api.tables(target.ref, target.database);
        const details = await Promise.all(
          tables.slice(0, 200).map((table) => api.table(target.ref, target.database, table.name)),
        );
        if (!live) return;
        setSchema({
          database: target.database,
          tables: details.map((detail: TableDetail) => ({
            name: detail.name,
            columns: (detail.columns ?? []).map((column) => column.name),
          })),
        });
      } catch {
        // Metadata is a convenience for the editor, never a reason to refuse a query.
        setSchema(null);
      }
    })();
    return () => {
      live = false;
    };
  }, [api, target.ref, target.database]);

  const run = async () => {
    if (!api) return;
    setBusy(true);
    setError(null);
    try {
      setRows(await api.query(target.ref, sql, 200));
    } catch (cause) {
      setRows(null);
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(false);
    }
  };

  const columns = useMemo(
    () => (rows?.columns ?? []).map((column, index) => ({ title: column.name, dataIndex: index, key: column.name })),
    [rows],
  );

  return (
    <Flex vertical gap={12}>
      <Alert type="info" showIcon message="SELECT, EXPLAIN and SHOW only" />
      <SqlEditor value={sql} onChange={setSql} schema={schema} height={200} />
      <Flex gap={8}>
        <Button type="primary" icon={<PlayCircleOutlined />} loading={busy} onClick={() => void run()}>
          Run
        </Button>
        <Button onClick={() => void navigator.clipboard.writeText(sql).then(() => message.success('copied'))}>
          Copy
        </Button>
      </Flex>
      {error ? <Alert type="error" showIcon message={error} /> : null}
      {rows ? (
        <Flex vertical gap={8}>
          <Typography.Text type="secondary">
            {rows.row_count} row{rows.row_count === 1 ? '' : 's'}
            {rows.truncated ? ' · truncated by the limit the API applied' : ''}
          </Typography.Text>
          <Table
            size="small"
            scroll={{ x: 'max-content' }}
            pagination={false}
            rowKey={(_, index) => String(index)}
            dataSource={(rows.values ?? []).map((row, index) => ({ key: index, row }))}
            columns={columns.map((column) => ({
              ...column,
              render: (_: unknown, record: { row: (string | null)[] }) => record.row[column.dataIndex] ?? '∅',
            }))}
          />
        </Flex>
      ) : null}
    </Flex>
  );
}

function QueueTab({ target, kind }: { target: Target; kind?: 'dml' | 'ddl' }) {
  const { api } = useSession();
  const [state, load] = useLoader<Proposal[]>();
  const [status, setStatus] = useState<string>('all');
  const [term, setTerm] = useState('');

  const fetch = useCallback(async () => {
    if (!api) return [];
    const page = await api.proposals({
      target: target.ref,
      kind,
      status: status === 'all' ? undefined : status,
      limit: 50,
    });
    return page.items;
  }, [api, target.ref, kind, status]);

  useEffect(() => {
    void load(fetch);
  }, [load, fetch]);

  // The API filters by status; it has no free-text query for proposals, so the title filter runs over
  // the page that came back. Filtering what is on screen is honest; pretending to search the queue is
  // not.
  const visible = state.kind === 'ready' ? state.value.filter((item) => matches(item, term)) : [];

  return (
    <Flex vertical gap={12}>
      <Flex gap={8} wrap align="center">
        <Input
          allowClear
          placeholder="Filter by title"
          value={term}
          onChange={(event) => setTerm(event.target.value)}
          style={{ maxWidth: 280 }}
          aria-label="Filter proposals by title"
        />
        <Segmented
          value={status}
          onChange={(value) => setStatus(String(value))}
          options={['all', 'verified', 'approved', 'applied', 'failed']}
        />
        <span style={{ flex: 1 }} />
        <Typography.Text type="secondary">
          {visible.length} of {state.kind === 'ready' ? state.value.length : 0}
        </Typography.Text>
      </Flex>

      <Async
        state={state}
        isEmpty={() => visible.length === 0}
        retry={() => void load(fetch)}
        empty={<Empty description="Nothing matches" image={Empty.PRESENTED_IMAGE_SIMPLE} />}
      >
        {() => (
          <Table
            size="small"
            rowKey="ref"
            pagination={false}
            scroll={{ x: 'max-content' }}
            dataSource={visible}
            columns={[
              {
                title: 'Title',
                dataIndex: 'title',
                render: (_: unknown, row: Proposal) => <Link to={`/proposals/${row.ref}`}>{row.title}</Link>,
              },
              { title: 'Status', dataIndex: 'status', render: (value: string) => <Tag>{value}</Tag> },
              { title: 'By', dataIndex: 'proposed_by', responsive: ['md'] },
              {
                title: 'Statement',
                dataIndex: 'query',
                responsive: ['lg'],
                render: (query: string) => (
                  <Typography.Text className="noxis-mono" ellipsis style={{ maxWidth: 320 }}>
                    {query}
                  </Typography.Text>
                ),
              },
            ]}
          />
        )}
      </Async>
    </Flex>
  );
}

function matches(proposal: Proposal, term: string): boolean {
  const needle = term.trim().toLowerCase();
  if (!needle) return true;
  return (
    proposal.title.toLowerCase().includes(needle) ||
    proposal.query.toLowerCase().includes(needle) ||
    proposal.proposed_by.toLowerCase().includes(needle)
  );
}
