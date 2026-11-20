import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  App as AntApp,
  Button,
  Card,
  Descriptions,
  Flex,
  Input,
  Space,
  Tag,
  Timeline,
  Typography,
} from 'antd';
import { CheckOutlined, CloseOutlined, RedoOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { Link, useParams } from 'react-router-dom';
import { useCan, useSession } from '../api/session';
import { Async, useLoader } from '../components/Async';
import type { JournalEntry, ProposalDetail } from '../api/types';

// One proposal, with the evidence beside the two buttons that matter. The console does not decide
// anything: it sends the same requests the CLI does and shows what the server answered.

const STATUS_COLOUR: Record<string, string> = {
  verified: 'green',
  partial: 'gold',
  failed: 'red',
  approved: 'blue',
  applied: 'green',
  rejected: 'default',
  applying: 'processing',
};

export function ProposalPage() {
  const { ref = '' } = useParams();
  const { api } = useSession();
  const { message, modal } = AntApp.useApp();
  const canApprove = useCan('approve');
  const canApply = useCan('apply');

  const [detail, loadDetail] = useLoader<ProposalDetail>();
  const [events, setEvents] = useState<JournalEntry[]>([]);
  const [busy, setBusy] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!api) return;
    await loadDetail(() => api.proposal(ref));
    setEvents(await api.events(ref));
  }, [api, ref, loadDetail]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const act = async (name: string, work: () => Promise<unknown>) => {
    setBusy(name);
    try {
      await work();
      await refresh();
      void message.success(`${name} done`);
    } catch (cause) {
      void message.error(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(null);
    }
  };

  const apply = (proposal: ProposalDetail) => {
    if (!proposal.irreversible) {
      void act('apply', () => api!.apply(ref));
      return;
    }
    // An irreversible change is acknowledged by typing its own name: a checkbox is a click, and this
    // is the one action in the product that cannot be taken back.
    let typed = '';
    modal.confirm({
      title: 'This change cannot be undone',
      content: (
        <Flex vertical gap={8}>
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            The rehearsal found no rollback. Type <Typography.Text code>{proposal.title}</Typography.Text> to
            confirm.
          </Typography.Paragraph>
          <Input onChange={(event) => (typed = event.target.value)} placeholder={proposal.title} />
        </Flex>
      ),
      okText: 'Apply anyway',
      okButtonProps: { danger: true },
      onOk: async () => {
        if (typed !== proposal.title) {
          void message.error('the title was not typed, so nothing was sent');
          return Promise.reject(new Error('not confirmed'));
        }
        await act('apply', () => api!.apply(ref, true));
      },
    });
  };

  return (
    <Flex vertical gap={16}>
      <Link to="/">← Instances</Link>
      <Async state={detail} retry={() => void refresh()}>
        {(proposal) => (
          <Flex vertical gap={16}>
            <Card>
              <Flex vertical gap={12}>
                <Flex align="center" gap={10} wrap>
                  <Typography.Title level={3} style={{ margin: 0 }}>
                    {proposal.title}
                  </Typography.Title>
                  <Tag>{proposal.kind}</Tag>
                  <Tag color={STATUS_COLOUR[proposal.status] ?? 'default'}>{proposal.status}</Tag>
                  {proposal.irreversible ? <Tag color="red">irreversible</Tag> : null}
                </Flex>
                <Descriptions size="small" column={{ xs: 1, sm: 2 }}>
                  <Descriptions.Item label="Proposed by">{proposal.proposed_by}</Descriptions.Item>
                  <Descriptions.Item label="Tier">{proposal.tier}</Descriptions.Item>
                  <Descriptions.Item label="Target">
                    <Link to={`/targets/${proposal.target_ref}`}>{proposal.target_ref}</Link>
                  </Descriptions.Item>
                  <Descriptions.Item label="Created">{proposal.created_at}</Descriptions.Item>
                  {proposal.algorithm ? (
                    <Descriptions.Item label="Algorithm">{proposal.algorithm}</Descriptions.Item>
                  ) : null}
                  {proposal.rows_affected !== undefined ? (
                    <Descriptions.Item label="Rows">{proposal.rows_affected}</Descriptions.Item>
                  ) : null}
                </Descriptions>

                <Typography.Text type="secondary">Statement</Typography.Text>
                <div className="noxis-sql-block">{proposal.query}</div>
                <Typography.Text type="secondary">Rollback</Typography.Text>
                {proposal.rollback ? (
                  <div className="noxis-sql-block">{proposal.rollback}</div>
                ) : (
                  <Alert type="warning" showIcon message="No rollback: this change is irreversible" />
                )}

                <Space wrap>
                  <Button
                    icon={<CheckOutlined />}
                    type="primary"
                    disabled={!canApprove || proposal.status === 'approved' || proposal.status === 'applied'}
                    loading={busy === 'approve'}
                    onClick={() => void act('approve', () => api!.decide(ref, true))}
                  >
                    Approve
                  </Button>
                  <Button
                    icon={<CloseOutlined />}
                    danger
                    disabled={!canApprove || proposal.status === 'applied'}
                    loading={busy === 'reject'}
                    onClick={() => void act('reject', () => api!.decide(ref, false))}
                  >
                    Reject
                  </Button>
                  <Button
                    icon={<RedoOutlined />}
                    loading={busy === 'rehearse'}
                    onClick={() => void act('rehearse', () => api!.rehearse(ref))}
                  >
                    Rehearse again
                  </Button>
                  <Button
                    icon={<ThunderboltOutlined />}
                    type="primary"
                    danger={proposal.irreversible}
                    disabled={!canApply || proposal.status !== 'approved'}
                    loading={busy === 'apply'}
                    onClick={() => apply(proposal)}
                  >
                    Apply
                  </Button>
                  {!canApprove || !canApply ? (
                    <Typography.Text type="secondary">
                      Your token does not hold {!canApprove ? 'approve' : ''}
                      {!canApprove && !canApply ? ' or ' : ''}
                      {!canApply ? 'apply' : ''}.
                    </Typography.Text>
                  ) : null}
                </Space>
              </Flex>
            </Card>

            {proposal.rehearsal ? (
              <Card title="The rehearsal">
                <Flex vertical gap={10}>
                  <Flex gap={8} wrap>
                    <Tag color={STATUS_COLOUR[proposal.rehearsal.status] ?? 'default'}>
                      {proposal.rehearsal.status}
                    </Tag>
                    <Tag>{proposal.rehearsal.tier}</Tag>
                    {proposal.rehearsal.irreversible ? <Tag color="red">irreversible</Tag> : null}
                  </Flex>
                  {proposal.rehearsal.note ? (
                    <Typography.Paragraph style={{ marginBottom: 0 }}>
                      {proposal.rehearsal.note}
                    </Typography.Paragraph>
                  ) : null}
                  {proposal.rehearsal.error ? (
                    <Alert type="error" showIcon message={proposal.rehearsal.error} />
                  ) : null}
                  {proposal.rehearsal.schema_diff ? (
                    <>
                      <Typography.Text type="secondary">Schema diff</Typography.Text>
                      <div className="noxis-sql-block">{proposal.rehearsal.schema_diff}</div>
                    </>
                  ) : null}
                  {proposal.rehearsal.row_diff ? (
                    <>
                      <Typography.Text type="secondary">Row diff</Typography.Text>
                      <div className="noxis-sql-block">{proposal.rehearsal.row_diff}</div>
                    </>
                  ) : null}
                </Flex>
              </Card>
            ) : null}

            <Card title="History">
              {events.length === 0 ? (
                <Typography.Text type="secondary">Nothing recorded yet.</Typography.Text>
              ) : (
                <Timeline
                  items={events.map((entry) => ({
                    color: 'var(--noxis-orange)',
                    children: (
                      <Flex vertical gap={2}>
                        <Typography.Text strong>{entry.kind}</Typography.Text>
                        <Typography.Text type="secondary">{entry.actor} · {entry.created_at}</Typography.Text>
                        {Object.keys(entry.detail ?? {}).length > 0 ? (
                          <Typography.Text className="noxis-mono" type="secondary">
                            {Object.entries(entry.detail)
                              .map(([key, value]) => `${key}=${value}`)
                              .join('  ')}
                          </Typography.Text>
                        ) : null}
                      </Flex>
                    ),
                  }))}
                />
              )}
            </Card>
          </Flex>
        )}
      </Async>
    </Flex>
  );
}
