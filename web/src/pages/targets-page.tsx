import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import { App as AntApp, Card, Drawer, Input, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useCreateTarget, useTargets } from '../api/queries';
import type { TargetValues } from '../forms/schemas';
import { driverOptions, targetDefaults, targetSchema, tlsOptions } from '../forms/schemas';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { Form } from '../components/form';
import { NumberInput, SelectInput } from '../components/inputs';
import { RenderWhen } from '../components/render-when';
import { TextInput } from '../components/text-input';
import type { Target } from '../api/types';

// The landing screen: one card per instance and a search that asks the server. The list is a single
// infinite query whose key holds the search term, so typing cannot race a second request into the
// table — the bug the hand-rolled loader had.

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
        <h1 id="instances-title" className="m-0 text-xl font-semibold tracking-tight">
          Instances
        </h1>
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
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {items.map((target) => (
            <TargetCard
              key={target.ref}
              target={target}
              onOpen={() => navigate(`/targets/${target.ref}`)}
            />
          ))}
        </div>
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

function TargetCard({ target, onOpen }: { target: Target; onOpen: () => void }) {
  return (
    <Card
      hoverable
      onClick={onOpen}
      tabIndex={0}
      role="link"
      aria-label={`Open ${target.name}`}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onOpen();
        }
      }}
      className="cursor-pointer border-line transition-transform hover:-translate-y-px hover:border-brand"
    >
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <span className="text-[15px] font-semibold">{target.name}</span>
          <span className="flex-1" />
          <Tag>{target.driver}</Tag>
        </div>
        <span className="font-mono text-xs text-muted">
          {target.username}@{target.host}:{target.port}/{target.database}
        </span>
        <div className="flex flex-wrap gap-1.5">
          <RenderWhen condition={target.archived_at}>
            <Tag>archived</Tag>
          </RenderWhen>
          <RenderWhen condition={target.server_version}>
            <Tag>{target.server_version}</Tag>
          </RenderWhen>
          <Tag>{target.has_password ? 'password sealed' : 'no password'}</Tag>
          <RenderWhen condition={target.last_probe_status}>
            <Tag>probe: {target.last_probe_status}</Tag>
          </RenderWhen>
        </div>
      </div>
    </Card>
  );
}

function CreateTargetForm({ onCreated }: { onCreated: (target: Target) => void }) {
  const create = useCreateTarget();

  return (
    <Form<TargetValues>
      schema={targetSchema}
      defaultValues={targetDefaults}
      onSubmit={async (values) => {
        const target = await create.mutateAsync({ ...values, options: {} });
        onCreated(target);
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
      <TextInput
        name="password"
        label="Password"
        password
        autoComplete="new-password"
        hint="Sealed with SECRET_KEY; never returned."
      />
      <RenderWhen condition={create.error}>
        <p role="alert" className="mb-3 text-sm text-red-600">
          {create.error instanceof Error ? create.error.message : String(create.error)}
        </p>
      </RenderWhen>
      <Form.Submit block>Register</Form.Submit>
    </Form>
  );
}
