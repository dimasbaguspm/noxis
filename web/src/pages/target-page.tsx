import { ApiOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { App as AntApp, Input, Segmented, Tabs, Tag } from 'antd';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Can } from '../api/permissions';
import { Breadcrumb } from '../components/breadcrumb';
import { Card } from '../components/card';
import { Link, useParams } from 'react-router-dom';
import {
  useArchiveTarget,
  useDatabases,
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
import { EditConnection } from '../components/edit-connection';
import { Panel, Tile, Tiles } from '../components/panel';
import { RenderWhen } from '../components/render-when';
import { ResultSet } from '../components/result-set';
import { StatusTag } from '../components/sql-block';
import { Statistic } from '../components/statistic';
import { Eyebrow, Heading, Text } from '../components/text';
import { SqlEditor } from '../editor/sql-editor';
import { TablesTab } from './tables-tab';
import type { Proposal, Target } from '../api/types';
import type { Schema } from '../editor/sql-editor';
import { formatWhen } from '../lib/when';
import { readDraft, writeDraft } from '../lib/draft';

// The instance page: what this server is, and the tabs an operator works in. ERD is absent on purpose
// — the API exposes columns and indexes, not foreign keys, and a diagram from half a graph is a
// confident lie.

type Tab = 'query' | 'tables' | 'changes';

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
                    <Can do="update:target">
                      <EditConnection target={target.data} />
                    </Can>
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
                        target.data.last_probe_at ? formatWhen(target.data.last_probe_at) : '—'
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
                    key: 'tables',
                    label: 'Tables',
                    children: <TablesTab ref_={ref} registered={target.data.database} />,
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
            ? `answered ${formatWhen(target.last_probe_at)}`
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
  // The last statement this instance was asked is a draft, kept in the browser: a reload should not
  // cost a query somebody was halfway through.
  const [sql, setSql] = useState(() => readDraft(ref_) ?? 'SELECT 1 AS one;');
  const run = useRunQuery(ref_);
  const schema = useSchema(ref_, database);
  const databases = useDatabases(ref_);

  const kick = () => run.mutate({ sql });

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
          onChange={(next) => {
            setSql(next);
            writeDraft(ref_, next);
          }}
          onRun={kick}
          schema={schema.data ?? null}
          schemas={databases.data ?? []}
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
        {run.data ? (
          <ResultSet
            rows={run.data}
            name={`${database}-${new Date().toISOString().slice(0, 19).replaceAll(':', '')}.csv`}
          />
        ) : null}
      </RenderWhen>
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
              key: 'kind',
              title: 'Kind',
              width: 90,
              render: (row) => (
                <Text size="xs" tone="muted" className="font-semibold uppercase">
                  {row.kind}
                </Text>
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
