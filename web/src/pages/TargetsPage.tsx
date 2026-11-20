import { useCallback, useEffect, useState } from 'react';
import { App as AntApp, Button, Card, Drawer, Empty, Flex, Input, InputNumber, Select, Tag, Typography } from 'antd';
import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useSession } from '../api/session';
import { Async, useLoader } from '../components/Async';
import { Field, TextField } from '../components/Field';
import { targetDefaults, targetSchema } from '../forms/schemas';
import type { TargetValues } from '../forms/schemas';
import { zodResolver } from '@hookform/resolvers/zod';
import { Controller, useForm } from 'react-hook-form';
import type { Target } from '../api/types';

// The landing screen: everything the operator has, one card each, searchable, with the one action a
// new deployment needs — registering the first instance.

export function TargetsPage() {
  const { api } = useSession();
  const { message } = AntApp.useApp();
  const navigate = useNavigate();
  const [state, load] = useLoader<Target[]>();
  const [search, setSearch] = useState('');
  const [creating, setCreating] = useState(false);
  const [cursor, setCursor] = useState<string | undefined>();

  const fetchPage = useCallback(
    async (term: string, next?: string) => {
      if (!api) return;
      const page = await api.targets({ q: term, limit: 24, cursor: next });
      setCursor(page.next);
      return page.items;
    },
    [api],
  );

  useEffect(() => {
    void load(async () => (await fetchPage('')) ?? []);
  }, [load, fetchPage]);

  // The search filters on the server; a 250ms debounce is enough to stop typing from being a DoS.
  useEffect(() => {
    const timer = setTimeout(() => {
      void load(async () => (await fetchPage(search)) ?? []);
    }, 250);
    return () => clearTimeout(timer);
  }, [search, load, fetchPage]);

  return (
    <Flex vertical gap={20}>
      <Flex align="center" gap={12} wrap>
        <Typography.Title level={3} style={{ margin: 0 }}>
          Instances
        </Typography.Title>
        <Tag>MySQL · MariaDB</Tag>
        <span style={{ flex: 1 }} />
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder="Search by name"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          style={{ maxWidth: 320 }}
          aria-label="Search instances"
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreating(true)}>
          New instance
        </Button>
      </Flex>

      <Async
        state={state}
        isEmpty={(items) => items.length === 0}
        retry={() => void load(async () => (await fetchPage(search)) ?? [])}
        empty={
          <Empty
            description={search ? `Nothing matches “${search}”` : 'No instance yet'}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button type="primary" onClick={() => setCreating(true)}>
              Register the first one
            </Button>
          </Empty>
        }
      >
        {(items) => (
          <Flex vertical gap={16}>
            <div className="noxis-card-grid">
              {items.map((target) => (
                <Card
                  key={target.ref}
                  className="noxis-target-card"
                  onClick={() => navigate(`/targets/${target.ref}`)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') navigate(`/targets/${target.ref}`);
                  }}
                  role="button"
                  tabIndex={0}
                  aria-label={`Open ${target.name}`}
                >
                  <Flex vertical gap={8}>
                    <Flex align="center" gap={8}>
                      <Typography.Text strong style={{ fontSize: 15 }}>
                        {target.name}
                      </Typography.Text>
                      <span style={{ flex: 1 }} />
                      <Tag>{target.driver}</Tag>
                    </Flex>
                    <Typography.Text type="secondary" className="noxis-mono">
                      {target.username}@{target.host}:{target.port}/{target.database}
                    </Typography.Text>
                    <Flex gap={6} wrap>
                      {target.archived_at ? <Tag>archived</Tag> : null}
                      {target.server_version ? <Tag>{target.server_version}</Tag> : null}
                      {target.has_password ? <Tag>password sealed</Tag> : <Tag>no password</Tag>}
                      {target.last_probe_status ? <Tag>probe: {target.last_probe_status}</Tag> : null}
                    </Flex>
                  </Flex>
                </Card>
              ))}
            </div>
            {cursor ? (
              <Button
                onClick={() =>
                  void load(async () => {
                    const next = await fetchPage(search, cursor);
                    const current = state.kind === 'ready' ? state.value : [];
                    return [...current, ...(next ?? [])];
                  })
                }
              >
                Load more
              </Button>
            ) : null}
          </Flex>
        )}
      </Async>

      <CreateTargetDrawer
        open={creating}
        onClose={() => setCreating(false)}
        onCreated={(target) => {
          setCreating(false);
          void message.success(`${target.name} registered`);
          navigate(`/targets/${target.ref}`);
        }}
      />
    </Flex>
  );
}

function CreateTargetDrawer({
  open,
  onClose,
  onCreated,
}: {
  open: boolean;
  onClose: () => void;
  onCreated: (target: Target) => void;
}) {
  const { api } = useSession();
  const [refused, setRefused] = useState<string | null>(null);
  const form = useForm<TargetValues>({ resolver: zodResolver(targetSchema), defaultValues: targetDefaults });
  const errors = form.formState.errors;

  const submit = form.handleSubmit(async (values) => {
    if (!api) return;
    setRefused(null);
    try {
      onCreated(await api.createTarget({ ...values, options: {} }));
      form.reset(targetDefaults);
    } catch (cause) {
      // The server's sentence, not a generic one: it names the field that is wrong.
      setRefused(cause instanceof Error ? cause.message : String(cause));
    }
  });

  return (
    <Drawer
      title="New instance"
      open={open}
      onClose={onClose}
      width={Math.min(460, window.innerWidth)}
      destroyOnHidden
      footer={
        <Flex justify="flex-end" gap={8}>
          <Button onClick={onClose}>Cancel</Button>
          <Button type="primary" loading={form.formState.isSubmitting} onClick={() => void submit()}>
            Register
          </Button>
        </Flex>
      }
    >
      <form onSubmit={submit}>
        <TextField control={form.control} name="name" label="Name" autoFocus placeholder="prod" error={errors.name?.message} />

        <Flex gap={12}>
          <div style={{ flex: 1 }}>
            <Field label="Engine">
              <Controller
                control={form.control}
                name="driver"
                render={({ field }) => (
                  <Select
                    {...field}
                    options={[
                      { value: 'mysql', label: 'MySQL' },
                      { value: 'mariadb', label: 'MariaDB' },
                    ]}
                  />
                )}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="TLS">
              <Controller
                control={form.control}
                name="tls_mode"
                render={({ field }) => (
                  <Select
                    {...field}
                    options={[
                      { value: 'preferred', label: 'Preferred' },
                      { value: 'verify', label: 'Verify' },
                      { value: 'skip-verify', label: 'Skip verify' },
                    ]}
                  />
                )}
              />
            </Field>
          </div>
        </Flex>

        <Flex gap={12}>
          <div style={{ flex: 3 }}>
            <TextField control={form.control} name="host" label="Host" placeholder="db.internal" error={errors.host?.message} />
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Port" error={errors.port?.message}>
              <Controller
                control={form.control}
                name="port"
                render={({ field }) => (
                  <InputNumber min={1} max={65535} style={{ width: '100%' }} value={field.value} onChange={(v) => field.onChange(v ?? 0)} />
                )}
              />
            </Field>
          </div>
        </Flex>

        <TextField control={form.control} name="database" label="Database" placeholder="app" error={errors.database?.message} />
        <TextField
          control={form.control}
          name="username"
          label="Username"
          placeholder="noxis"
          autoComplete="off"
          error={errors.username?.message}
        />
        <TextField
          control={form.control}
          name="password"
          label="Password"
          password
          autoComplete="new-password"
          hint="Sealed with SECRET_KEY; never returned."
        />

        {refused ? <Typography.Text type="danger">{refused}</Typography.Text> : null}
      </form>
    </Drawer>
  );
}
