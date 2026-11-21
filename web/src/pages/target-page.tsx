import {
  ApiOutlined,
  DeleteOutlined,
  EditOutlined,
  MoreOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import { App as AntApp, Dropdown, Input, Tabs, Tag } from 'antd';
import type { MenuProps } from 'antd';
import { useState } from 'react';
import type { ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { usePermissions } from '../api/permissions';
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
import { RenderWhen } from '../components/render-when';
import { ResultSet } from '../components/result-set';
import { StatusTag } from '../components/sql-block';
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

type Tab = 'details' | 'query' | 'tables' | 'changes';

// The record comes first, the work after it, and each tab is a route of its own —
// /targets/:ref/changes — so a tab can be linked, bookmarked, and survives a reload. The bare
// `/targets/:ref` opens the first tab.
const TABS: Tab[] = ['details', 'query', 'tables', 'changes'];

export function TargetPage() {
  const { ref = '', tab: asked } = useParams();
  const tab: Tab = TABS.includes(asked as Tab) ? (asked as Tab) : TABS[0];
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
              <InstanceHeader target={target.data} onArchive={archiveIt} />

              <Tabs
                activeKey={tab}
                onChange={(key) => navigate(`/targets/${ref}/${key}`)}
                items={[
                  {
                    key: 'details',
                    label: 'Details',
                    children: <DetailsTab target={target.data} tableCount={tables.data?.length} />,
                  },
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

// The Details tab is the whole record in the shape that reads as a record: a definition list under an
// "Overview" head, because the tab already says which tab it is. The header used to shout these as
// numbers; the page above stays a name and a connectivity check.
function DetailsTab({ target, tableCount }: { target: Target; tableCount?: number }) {
  return (
    <Card title="Overview">
      <dl className="m-0 grid grid-cols-1 gap-x-8 gap-y-4 sm:grid-cols-2 lg:grid-cols-3">
        <Fact label="Version" value={target.server_version ?? 'not probed'} mono />
        <Fact label="Host" value={`${target.username}@${target.host}`} mono />
        <Fact label="Schema" value={target.database} mono />
        <Fact label="Port" value={target.port} mono />
        <Fact label="TLS" value={target.tls_mode} />
        <Fact label="Tables" value={tableCount ?? 'counting'} />
        <Fact
          label="Last answer"
          value={
            target.last_probe_at
              ? `${target.last_probe_status ?? 'answered'} · ${formatWhen(target.last_probe_at)}`
              : 'never probed'
          }
        />
        <Fact label="Registered by" value={target.created_by} />
        <Fact label="Registered" value={formatWhen(target.created_at)} />
        <Fact label="Password" value={target.has_password ? 'stored, sealed' : 'none'} />
        <Fact label="Ref" value={target.ref} mono />
      </dl>
    </Card>
  );
}

/** One row of the definition list: label above, value under it, monospaced when it is an identifier. */
function Fact({ label, value, mono = false }: { label: string; value: ReactNode; mono?: boolean }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt>
        <Eyebrow>{label}</Eyebrow>
      </dt>
      <dd className="m-0 flex min-w-0">
        <Text size="sm" mono={mono} className="truncate">
          {value}
        </Text>
      </dd>
    </div>
  );
}

// The header carries the two facts an operator reads before touching anything — which server answered
// and whether it still does — and one menu for everything that can be done to the instance. The rest
// of the record is the Details tab; a header that repeats it is a header nobody finishes reading.
function InstanceHeader({ target, onArchive }: { target: Target; onArchive: () => void }) {
  const { message } = AntApp.useApp();
  const probe = useProbeTarget();
  const { allows } = usePermissions();
  const [editing, setEditing] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const ok = target.last_probe_status === 'ok';

  // Archive is the destructive one and says so with red, inside the menu rather than beside Edit where
  // a mis-aimed click used to be one pixel away from "the instance leaves the list".
  const items: MenuProps['items'] = [
    ...(allows('update:target')
      ? [{ key: 'edit', icon: <EditOutlined />, label: 'Edit connection' }]
      : []),
    ...(allows('delete:target') && !target.archived_at
      ? [{ key: 'archive', icon: <DeleteOutlined />, label: 'Archive', danger: true }]
      : []),
  ];

  return (
    <>
      <Card>
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
            <Heading level={1} size={2} id="target-title">
              {target.name}
            </Heading>
            <EngineMark engine={target.driver} size={16} />
            <RenderWhen condition={target.archived_at}>
              <Tag className="m-0">archived</Tag>
            </RenderWhen>
            <Text size="sm" tone="muted" mono className="truncate">
              {target.server_version ?? 'version unknown'}
            </Text>
            <div className="flex items-center gap-1.5">
              <RenderWhen condition={target.last_probe_status}>
                <Tag color={ok ? 'green' : 'red'} className="m-0">
                  {ok ? 'reachable' : 'unreachable'}
                </Tag>
              </RenderWhen>
              <RenderWhen condition={!target.last_probe_status}>
                <Tag className="m-0">never probed</Tag>
              </RenderWhen>
              <Text size="xs" tone="muted">
                {target.last_probe_at ? formatWhen(target.last_probe_at) : 'not contacted yet'}
              </Text>
            </div>
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
            <RenderWhen condition={items.length > 0}>
              <Dropdown
                trigger={['click']}
                placement="bottomRight"
                menu={{
                  items,
                  onClick: ({ key }) => (key === 'edit' ? setEditing(true) : onArchive()),
                }}
              >
                <AppButton icon={<MoreOutlined />} aria-label="Instance actions" />
              </Dropdown>
            </RenderWhen>
          </div>
          <RenderWhen condition={failure}>
            <Alert tone="error" title="The instance did not answer" description={failure} />
          </RenderWhen>
        </div>
      </Card>
      <EditConnection target={target} open={editing} onClose={() => setEditing(false)} />
    </>
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
          // The rows are a result, so they get a result card: the answer in its own frame, on the same
          // DataTable as every other table in the console, instead of floating under the editor.
          <Card title="Result">
            <ResultSet
              rows={run.data}
              name={`${database}-${new Date().toISOString().slice(0, 19).replaceAll(':', '')}.csv`}
            />
          </Card>
        ) : null}
      </RenderWhen>
    </div>
  );
}

/** The states a proposal moves through, in order: what the status column can be narrowed to. */
const PROPOSAL_STATUSES = [
  'draft',
  'verified',
  'approved',
  'applying',
  'applied',
  'failed',
  'rejected',
];

function Changes({ target, schema }: { target: string; schema: Schema | null }) {
  const navigate = useNavigate();
  const [term, setTerm] = useState('');
  // Kind and status are filtered in the table's own header — a column that can be narrowed says so
  // where the operator is already reading. The title search stays above, where free text belongs.
  const proposals = useProposals({ target, limit: 50 });
  const items = proposals.data?.items ?? [];
  const needle = term.trim().toLowerCase();
  const visible = needle
    ? items.filter((item) =>
        [item.title, item.query, item.proposed_by].some((field) =>
          field.toLowerCase().includes(needle),
        ),
      )
    : items;

  const open = (ref: string) => navigate(`/targets/${target}/changes/${ref}`);

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
        <span className="flex-1" />
        <CreateProposal target={target} schema={schema} />
      </div>

      <Card title="All changes" extra={`${visible.length} of ${items.length}`} bodyClassName="p-2">
        <AsyncState
          loading={proposals.isLoading}
          error={proposals.error}
          empty={visible.length === 0}
          emptyDescription={needle ? 'Nothing matches that title' : 'No changes yet'}
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
                    to={`/targets/${target}/changes/${row.ref}`}
                    className="text-brand-ink hover:underline"
                  >
                    {row.title}
                  </Link>
                ),
              },
              {
                key: 'kind',
                title: 'Kind',
                width: 120,
                filters: [
                  { text: 'DDL', value: 'ddl' },
                  { text: 'DML', value: 'dml' },
                ],
                render: (row) => (
                  <Text size="xs" tone="muted" className="font-semibold uppercase">
                    {row.kind}
                  </Text>
                ),
              },
              {
                key: 'status',
                title: 'Status',
                width: 170,
                filters: PROPOSAL_STATUSES.map((status) => ({ text: status, value: status })),
                render: (row) => <StatusTag status={row.status} />,
              },
              {
                key: 'updated_at',
                title: 'Last update',
                width: 140,
                render: (row) => (
                  <Text size="xs" tone="muted">
                    {formatWhen(row.updated_at)}
                  </Text>
                ),
              },
              { key: 'proposed_by', title: 'Proposed by', showFrom: 'md' },
            ]}
          />
        </AsyncState>
      </Card>
    </div>
  );
}
