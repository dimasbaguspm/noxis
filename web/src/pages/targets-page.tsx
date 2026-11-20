import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import { App as AntApp, Drawer, Input, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateTarget, useTargets, useTestConnection } from '../api/queries';
import { driverOptions, targetDefaults, targetSchema, tlsOptions } from '../forms/schemas';
import type { TargetValues } from '../forms/schemas';
import { Alert } from '../components/alert';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { DataTable } from '../components/data-table';
import { Form } from '../components/form';
import { NumberInput, SelectInput } from '../components/inputs';
import { RenderWhen } from '../components/render-when';
import { Heading, Text } from '../components/text';
import { TextInput } from '../components/text-input';
import type { Target } from '../api/types';

// The landing screen: one row per instance and a search that asks the server. The list is a single
// infinite query whose key holds the search term, so typing cannot race a second request into the
// table. The name column is pinned, so a narrow window scrolls the facts and keeps the identity.

const FORM_ID = 'create-target';

function useDebounced(value: string, delay: number): string {
  const [settled, setSettled] = useState(value);
  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(value), delay);
    return () => window.clearTimeout(timer);
  }, [value, delay]);
  return settled;
}

export function TargetsPage() {
  const navigate = useNavigate();
  const { message } = AntApp.useApp();
  const [term, setTerm] = useState('');
  const [creating, setCreating] = useState(false);
  const search = useDebounced(term, 250);
  const targets = useTargets(search);
  const items = targets.data?.pages.flatMap((page) => page.items) ?? [];

  return (
    <section className="flex flex-col gap-5" aria-labelledby="instances-title">
      <div className="flex flex-wrap items-center gap-3">
        <Heading level={1} id="instances-title">
          Instances
        </Heading>
        <Tag className="hidden sm:inline">MySQL · MariaDB</Tag>
        <span className="flex-1" />
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder="Search by name"
          value={term}
          onChange={(event) => setTerm(event.target.value)}
          aria-label="Search instances"
          className="max-w-xs"
        />
        <AppButton intent="primary" icon={<PlusOutlined />} onClick={() => setCreating(true)}>
          New instance
        </AppButton>
      </div>

      <AsyncState
        loading={targets.isLoading}
        error={targets.error}
        empty={items.length === 0}
        emptyDescription={search ? `Nothing matches “${search}”` : 'No instance yet'}
        emptyAction={
          <AppButton intent="primary" onClick={() => setCreating(true)}>
            Register the first one
          </AppButton>
        }
        onRetry={() => void targets.refetch()}
      >
        <DataTable<Target>
          rows={items}
          rowKey={(row) => row.ref}
          minWidth={860}
          sticky
          onRow={(row) => ({
            onClick: () => navigate(`/targets/${row.ref}`),
            onKeyDown: (event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                navigate(`/targets/${row.ref}`);
              }
            },
            tabIndex: 0,
            role: 'link',
            'aria-label': `Open ${row.name}`,
            className: 'cursor-pointer',
          })}
          columns={[
            {
              key: 'name',
              title: 'Name',
              fixed: 'left',
              width: 220,
              render: (row) => (
                <span className="flex items-center gap-2">
                  <span aria-hidden="true" className="h-2 w-2 shrink-0 rounded-sm bg-brand" />
                  <span className="font-semibold text-ink">{row.name}</span>
                  <RenderWhen condition={row.archived_at}>
                    <Tag className="m-0">archived</Tag>
                  </RenderWhen>
                </span>
              ),
            },
            {
              key: 'driver',
              title: 'Engine',
              width: 100,
              render: (row) => <Tag>{row.driver}</Tag>,
            },
            {
              key: 'host',
              title: 'Host',
              width: 220,
              render: (row) => (
                <Text mono size="sm" tone="muted">
                  {row.host}:{row.port}
                </Text>
              ),
            },
            {
              key: 'database',
              title: 'Database',
              width: 150,
              render: (row) => (
                <Text mono size="sm">
                  {row.database}
                </Text>
              ),
            },
            {
              key: 'server_version',
              title: 'Version',
              width: 120,
              render: (row) => (
                <Text mono size="sm" tone="muted">
                  {row.server_version ?? '—'}
                </Text>
              ),
            },
            {
              key: 'last_probe_status',
              title: 'Probe',
              width: 120,
              render: (row) =>
                row.last_probe_status ? (
                  <Tag color={row.last_probe_status === 'ok' ? 'green' : 'red'} className="m-0">
                    {row.last_probe_status}
                  </Tag>
                ) : (
                  <Text tone="muted" size="sm">
                    never
                  </Text>
                ),
            },
            {
              key: 'open',
              title: '',
              width: 90,
              fixed: 'right',
              align: 'right',
              render: (row) => (
                <AppButton
                  size="small"
                  onClick={(event) => {
                    event.stopPropagation();
                    navigate(`/targets/${row.ref}`);
                  }}
                >
                  Open
                </AppButton>
              ),
            },
          ]}
        />
        <RenderWhen condition={targets.hasNextPage}>
          <div className="mt-4 flex justify-center">
            <AppButton
              onClick={() => void targets.fetchNextPage()}
              loading={targets.isFetchingNextPage}
            >
              Load more
            </AppButton>
          </div>
        </RenderWhen>
      </AsyncState>

      <Drawer
        title="New instance"
        open={creating}
        onClose={() => setCreating(false)}
        width={Math.min(460, window.innerWidth)}
        destroyOnHidden
        footer={
          <div className="flex justify-end gap-2">
            <AppButton onClick={() => setCreating(false)}>Cancel</AppButton>
            <AppButton intent="primary" htmlType="submit" form={FORM_ID}>
              Register
            </AppButton>
          </div>
        }
      >
        <CreateTargetForm
          onCreated={(target) => {
            setCreating(false);
            void message.success(`${target.name} registered`);
            navigate(`/targets/${target.ref}`);
          }}
        />
      </Drawer>
    </section>
  );
}

// The test dials a connection, not a registration: the API refuses a field it did not ask for, and a
// name is the row's business rather than the socket's.
function connectionOf(values: TargetValues) {
  return {
    driver: values.driver,
    host: values.host,
    port: values.port,
    database: values.database,
    username: values.username,
    password: values.password,
    tls_mode: values.tls_mode,
    options: {},
  };
}

function CreateTargetForm({ onCreated }: { onCreated: (target: Target) => void }) {
  const create = useCreateTarget();
  const test = useTestConnection();
  // A refused test keeps the values that failed: "register anyway" is the operator's decision, not a
  // second submit that would dial the same dead host again.
  const [blocked, setBlocked] = useState<{ values: TargetValues; message: string } | null>(null);

  const send = async (values: TargetValues) => {
    onCreated(await create.mutateAsync({ ...values, options: {} }));
  };

  return (
    <Form<TargetValues>
      id={FORM_ID}
      schema={targetSchema}
      defaultValues={targetDefaults}
      onSubmit={async (values) => {
        setBlocked(null);
        try {
          await test.mutateAsync(connectionOf(values));
        } catch (error) {
          setBlocked({ values, message: error instanceof Error ? error.message : String(error) });
          return;
        }
        await send(values);
      }}
    >
      <TextInput name="name" label="Name" autoFocus placeholder="prod" />
      <div className="flex gap-3">
        <div className="flex-1">
          <SelectInput name="driver" label="Engine" options={driverOptions} />
        </div>
        <div className="flex-1">
          <SelectInput name="tls_mode" label="TLS" options={tlsOptions} />
        </div>
      </div>
      <div className="flex gap-3">
        <div className="flex-[3]">
          <TextInput name="host" label="Host" placeholder="db.internal" />
        </div>
        <div className="flex-1">
          <NumberInput name="port" label="Port" min={1} max={65535} />
        </div>
      </div>
      <TextInput name="database" label="Database" placeholder="app" />
      <TextInput name="username" label="Username" placeholder="noxis" autoComplete="off" />
      <TextInput name="password" label="Password" password autoComplete="new-password" />
      <RenderWhen condition={blocked}>
        <Alert
          tone="warning"
          className="mb-3"
          title="The instance did not answer"
          description={blocked?.message}
          action={
            <AppButton
              size="small"
              loading={create.isPending}
              onClick={() => blocked && void send(blocked.values)}
            >
              Register anyway
            </AppButton>
          }
        />
      </RenderWhen>
    </Form>
  );
}
