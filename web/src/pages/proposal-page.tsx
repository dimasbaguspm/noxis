import { CheckOutlined, CloseOutlined, RedoOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { Alert, App as AntApp, Card, Descriptions, Flex, Input, Tag, Timeline } from 'antd';
import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useSession } from '../api/session-context';
import { Can, usePermissions } from '../api/permissions';
import { useEvents, useProposal, useProposalActions } from '../api/queries';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { RenderWhen } from '../components/render-when';
import { SqlBlock, StatusTag } from '../components/sql-block';

// One proposal with its evidence beside the two buttons that matter. The console decides nothing: it
// sends the same requests the CLI does and shows what the server answered.

export function ProposalPage() {
  const { ref = '' } = useParams();
  const { identity } = useSession();
  const { modal, message } = AntApp.useApp();
  const proposal = useProposal(ref);
  const events = useEvents(ref);
  const actions = useProposalActions(ref);
  const { can } = usePermissions();
  const [confirming, setConfirming] = useState<string | null>(null);

  const found = proposal.data;
  const busy = actions.decide.isPending || actions.rehearse.isPending || actions.apply.isPending;

  const run = async (work: () => Promise<unknown>, done: string) => {
    try {
      await work();
      void message.success(done);
    } catch (cause) {
      void message.error(cause instanceof Error ? cause.message : String(cause));
    }
  };

  const apply = () => {
    if (!found) return;
    if (!found.irreversible) {
      void run(() => actions.apply.mutateAsync(false), 'applied');
      return;
    }
    // An irreversible change is acknowledged by typing its own title: a checkbox is a click, and this
    // is the one action in the product that cannot be taken back.
    let typed = '';
    modal.confirm({
      title: 'This change cannot be undone',
      okText: 'Apply anyway',
      okButtonProps: { danger: true },
      content: (
        <Flex vertical gap={8}>
          <p className="m-0">
            The rehearsal found no rollback. Type <strong>{found.title}</strong> to confirm.
          </p>
          <Input
            onChange={(event) => {
              typed = event.target.value;
            }}
            aria-label="type the proposal title to confirm"
          />
        </Flex>
      ),
      onOk: async () => {
        if (typed !== found.title) {
          void message.error('the title was not typed, so nothing was sent');
          return Promise.reject(new Error('not confirmed'));
        }
        await run(() => actions.apply.mutateAsync(true), 'applied');
      },
    });
  };

  return (
    <section className="flex flex-col gap-5" aria-labelledby="proposal-title">
      <Link to="/" className="w-fit text-sm text-brand hover:underline">
        ← Instances
      </Link>

      <AsyncState
        loading={proposal.isLoading}
        error={proposal.error}
        onRetry={() => void proposal.refetch()}
      >
        <RenderWhen condition={found}>
          {found ? (
            <div className="flex flex-col gap-4">
              <Card>
                <div className="flex flex-col gap-3">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <h1 id="proposal-title" className="m-0 text-xl font-semibold">
                      {found.title}
                    </h1>
                    <Tag>{found.kind}</Tag>
                    <StatusTag status={found.status} />
                    <RenderWhen condition={found.irreversible}>
                      <Tag color="red">irreversible</Tag>
                    </RenderWhen>
                  </div>

                  <Descriptions size="small" column={{ xs: 1, sm: 2 }}>
                    <Descriptions.Item label="Proposed by">{found.proposed_by}</Descriptions.Item>
                    <Descriptions.Item label="Tier">{found.tier}</Descriptions.Item>
                    <Descriptions.Item label="Instance">
                      <Link
                        to={`/targets/${found.target_ref}`}
                        className="text-brand hover:underline"
                      >
                        open
                      </Link>
                    </Descriptions.Item>
                    <Descriptions.Item label="Created">{found.created_at}</Descriptions.Item>
                  </Descriptions>

                  <SqlBlock sql={found.query} label="statement" />
                  {found.rollback ? (
                    <SqlBlock sql={found.rollback} label="rollback" />
                  ) : (
                    <Alert
                      type="warning"
                      showIcon
                      message="No rollback: this change is irreversible"
                    />
                  )}

                  <div className="flex flex-wrap items-center gap-2">
                    <AppButton
                      intent="primary"
                      icon={<CheckOutlined />}
                      disabled={
                        !can['review:proposal'] || ['approved', 'applied'].includes(found.status)
                      }
                      loading={busy}
                      onClick={() =>
                        void run(() => actions.decide.mutateAsync({ approve: true }), 'approved')
                      }
                    >
                      Approve
                    </AppButton>
                    <AppButton
                      intent="danger"
                      icon={<CloseOutlined />}
                      disabled={!can['review:proposal'] || found.status === 'applied'}
                      onClick={() => {
                        setConfirming(found.ref);
                        void run(
                          () => actions.decide.mutateAsync({ approve: false }),
                          'rejected',
                        ).finally(() => setConfirming(null));
                      }}
                    >
                      Reject
                    </AppButton>
                    <AppButton
                      icon={<RedoOutlined />}
                      loading={actions.rehearse.isPending}
                      onClick={() => void run(() => actions.rehearse.mutateAsync(), 'rehearsed')}
                    >
                      Rehearse again
                    </AppButton>
                    <AppButton
                      intent="primary"
                      icon={<ThunderboltOutlined />}
                      disabled={!can['execute:proposal'] || found.status !== 'approved'}
                      onClick={apply}
                    >
                      Apply
                    </AppButton>
                    <Can
                      do="review:proposal"
                      fallback={
                        <span className="text-xs text-muted">
                          {identity?.name} cannot review this change
                        </span>
                      }
                    >
                      <Can
                        do="execute:proposal"
                        fallback={
                          <span className="text-xs text-muted">apply needs the apply scope</span>
                        }
                      >
                        <span />
                      </Can>
                    </Can>
                  </div>
                  <RenderWhen condition={confirming}>
                    <span aria-live="polite" className="text-xs text-muted">
                      rejecting…
                    </span>
                  </RenderWhen>
                </div>
              </Card>

              <RenderWhen condition={found.rehearsal}>
                {found.rehearsal ? (
                  <Card title="The rehearsal">
                    <div className="flex flex-col gap-2.5">
                      <div className="flex flex-wrap gap-2">
                        <StatusTag status={found.rehearsal.status} />
                        <Tag>{found.rehearsal.tier}</Tag>
                        <RenderWhen condition={found.rehearsal.irreversible}>
                          <Tag color="red">irreversible</Tag>
                        </RenderWhen>
                      </div>
                      <RenderWhen condition={found.rehearsal.note}>
                        <p className="m-0 text-sm">{found.rehearsal.note}</p>
                      </RenderWhen>
                      <RenderWhen condition={found.rehearsal.error}>
                        <Alert type="error" showIcon message={found.rehearsal.error} />
                      </RenderWhen>
                      <RenderWhen condition={found.rehearsal.schema_diff}>
                        <SqlBlock sql={found.rehearsal.schema_diff ?? ''} label="schema diff" />
                      </RenderWhen>
                      <RenderWhen condition={found.rehearsal.row_diff}>
                        <SqlBlock sql={found.rehearsal.row_diff ?? ''} label="row diff" />
                      </RenderWhen>
                    </div>
                  </Card>
                ) : null}
              </RenderWhen>

              <Card title="History" loading={events.isLoading}>
                <RenderWhen
                  condition={(events.data ?? []).length > 0}
                  fallback={<span className="text-sm text-muted">Nothing recorded yet.</span>}
                >
                  <Timeline
                    items={(events.data ?? []).map((entry) => ({
                      color: 'var(--color-brand)',
                      children: (
                        <div className="flex flex-col gap-0.5">
                          <span className="font-medium capitalize">{entry.kind}</span>
                          <span className="text-xs text-muted">
                            {entry.actor} · {entry.created_at}
                          </span>
                          {Object.keys(entry.detail ?? {}).length > 0 ? (
                            <span className="font-mono text-xs text-muted">
                              {Object.entries(entry.detail)
                                .map(([key, value]) => `${key}=${value}`)
                                .join('  ')}
                            </span>
                          ) : null}
                        </div>
                      ),
                    }))}
                  />
                </RenderWhen>
              </Card>
            </div>
          ) : null}
        </RenderWhen>
      </AsyncState>
    </section>
  );
}
