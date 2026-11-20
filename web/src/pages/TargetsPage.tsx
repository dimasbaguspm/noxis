import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  App as AntApp,
  Button,
  Card,
  Drawer,
  Empty,
  Flex,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Tag,
  Typography,
} from 'antd';
import { PlusOutlined, SearchOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useSession } from '../api/session';
import { Async, useLoader } from '../components/Async';
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

type FormValues = {
  name: string;
  driver: string;
  host: string;
  port: number;
  database: string;
  username: string;
  password?: string;
  tls_mode: string;
};

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
  const [form] = Form.useForm<FormValues>();
  const [busy, setBusy] = useState(false);
  const [refused, setRefused] = useState<string | null>(null);
  const initial = useMemo<Partial<FormValues>>(
    () => ({ driver: 'mysql', port: 3306, tls_mode: 'preferred' }),
    [],
  );

  const submit = async (values: FormValues) => {
    if (!api) return;
    setBusy(true);
    setRefused(null);
    try {
      onCreated(await api.createTarget({ ...values, options: {} }));
      form.resetFields();
    } catch (cause) {
      // The server's sentence, not a generic one: it names the field that is wrong.
      setRefused(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      title="New instance"
      open={open}
      onClose={onClose}
      width={Math.min(480, window.innerWidth)}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" initialValues={initial} onFinish={submit} requiredMark={false}>
        <Form.Item name="name" label="Name" rules={[{ required: true }]}>
          <Input autoFocus placeholder="prod" />
        </Form.Item>
        <Flex gap={12}>
          <Form.Item name="driver" label="Engine" style={{ flex: 1 }} rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'mysql', label: 'MySQL 8' },
                { value: 'mariadb', label: 'MariaDB 11' },
              ]}
            />
          </Form.Item>
          <Form.Item name="tls_mode" label="TLS" style={{ flex: 1 }}>
            <Select
              options={[
                { value: 'preferred', label: 'preferred' },
                { value: 'required', label: 'required' },
                { value: 'skip-verify', label: 'skip verify' },
                { value: 'disabled', label: 'disabled' },
              ]}
            />
          </Form.Item>
        </Flex>
        <Flex gap={12}>
          <Form.Item name="host" label="Host" style={{ flex: 3 }} rules={[{ required: true }]}>
            <Input placeholder="db.internal" />
          </Form.Item>
          <Form.Item name="port" label="Port" style={{ flex: 1 }} rules={[{ required: true }]}>
            <InputNumber min={1} max={65535} style={{ width: '100%' }} />
          </Form.Item>
        </Flex>
        <Form.Item name="database" label="Database" rules={[{ required: true }]}>
          <Input placeholder="app" />
        </Form.Item>
        <Form.Item name="username" label="Username" rules={[{ required: true }]}>
          <Input placeholder="noxis" autoComplete="off" />
        </Form.Item>
        <Form.Item
          name="password"
          label="Password"
          extra="Sealed with SECRET_KEY and never returned by the API."
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>

        {refused ? <Typography.Paragraph type="danger">{refused}</Typography.Paragraph> : null}

        <Space>
          <Button type="primary" htmlType="submit" loading={busy}>
            Register
          </Button>
          <Button onClick={onClose}>Cancel</Button>
        </Space>
      </Form>
    </Drawer>
  );
}
