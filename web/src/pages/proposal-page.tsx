import { CheckOutlined, CloseOutlined, RedoOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { App as AntApp, Flex, Input, Tag, Timeline } from 'antd';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useSession } from '../api/session-context';
import { Can, usePermissions } from '../api/permissions';
import { useEvents, useProposal, useProposalActions, useTarget } from '../api/queries';
import { Alert } from '../components/alert';
import { AppButton } from '../components/app-button';
import { AsyncState } from '../components/async-state';
import { Breadcrumb } from '../components/breadcrumb';
import { Card } from '../components/card';
import { RenderWhen } from '../components/render-when';
import { StatusTag } from '../components/sql-block';
import { Eyebrow, Heading, Text } from '../components/text';
import { Verdict, verdictOf } from '../components/verdict';
import { SchemaDiff } from '../editor/diff-editor';
import { SqlEditor } from '../editor/sql-editor';
import { formatWhen } from '../lib/when';

// One proposal, with its evidence, under the instance it belongs to. The console decides nothing: it
// sends the same requests the CLI does and shows what the server answered.
//
// The page reads in the order a reviewer actually works: what was proven (the verdict), what it is (the
// statement), what undoes it (the rollback), how it was proven (the diffs), and only then the decision —
// which sits in the aside so the buttons are where the reading ends, not where it began.

export function ProposalPage() {
  const { ref = '', proposalRef = '' } = useParams();
  const target = useTarget(ref);
  const { identity } = useSession();
  const { modal, message } = AntApp.useApp();
  const proposal = useProposal(proposalRef);
  const events = useEvents(proposalRef);
  const actions = useProposalActions(proposalRef);
  const { can } = usePermissions();
  const [confirming, setConfirming] = useState<string | null>(null);

  const found = proposal.data;
  const busy = actions.decide.isPending || actions.rehearse.isPending || actions.apply.isPending;
  const verdict = found ? verdictOf(found) : undefined;

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
          <Text as="p">
            The rehearsal found no rollback. Type <strong>{found.title}</strong> to confirm.
          </Text>
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
      <Breadcrumb
        items={[
          { label: 'Instances', to: '/' },
          { label: target.data?.name ?? 'Instance', to: `/targets/${ref}` },
          { label: found?.title ?? 'Proposal' },
        ]}
      />

      <AsyncState
        loading={proposal.isLoading}
        error={proposal.error}
        onRetry={() => void proposal.refetch()}
      >
        <RenderWhen condition={found}>
          {found ? (
            <div className="flex flex-col gap-4">
              <div className="flex flex-wrap items-center gap-2">
                <Heading level={1} id="proposal-title">
                  {found.title}
                </Heading>
                <StatusTag status={found.status} />
                <Tag className="m-0 uppercase">{found.kind}</Tag>
                <RenderWhen condition={found.irreversible}>
                  <Tag color="red" className="m-0">
                    irreversible
                  </Tag>
                </RenderWhen>
              </div>

              {/* The verdict comes before the evidence at every width: what was proven is what a
                  reviewer decides with, and the diffs below are how it was proven. */}
              <RenderWhen condition={verdict}>
                {verdict ? <Verdict verdict={verdict} /> : null}
              </RenderWhen>

              <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_340px] lg:items-start">
                <div className="flex min-w-0 flex-col gap-4">
                  <Card
                    title="Change"
                    extra={`proposed by ${found.proposed_by} · ${formatWhen(found.created_at)}`}
                    bodyClassName="flex flex-col gap-3 p-4"
                  >
                    <div>
                      <Eyebrow className="mb-1.5 block">statement</Eyebrow>
                      <div className="overflow-hidden rounded-lg border border-line">
                        <SqlEditor value={found.query} height={140} readOnly />
                      </div>
                    </div>

                    <RenderWhen condition={found.note}>
                      <Text as="p" size="sm" tone="muted">
                        {found.note}
                      </Text>
                    </RenderWhen>
                  </Card>

                  <Card title="Rollback" bodyClassName="p-4">
                    {found.rollback ? (
                      <div className="overflow-hidden rounded-lg border border-line">
                        <SqlEditor value={found.rollback} height={120} readOnly />
                      </div>
                    ) : (
                      <Alert tone="warning" title="No rollback: this change is irreversible" />
                    )}
                  </Card>

                  <RenderWhen condition={found.rehearsal}>
                    {found.rehearsal ? (
                      <Card
                        title="Evidence"
                        extra={formatWhen(found.rehearsal.created_at)}
                        bodyClassName="flex flex-col gap-3 p-4"
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <StatusTag status={found.rehearsal.status} />
                          <Tag className="m-0">{found.rehearsal.tier}</Tag>
                          <RenderWhen condition={found.rehearsal.irreversible}>
                            <Tag color="red" className="m-0">
                              irreversible
                            </Tag>
                          </RenderWhen>
                          <span className="flex-1" />
                          <Text size="xs" tone="muted">
                            {found.rehearsal.duration_ms ?? 0} ms
                          </Text>
                        </div>
                        <RenderWhen condition={found.rehearsal.note}>
                          <Text as="p" size="sm">
                            {found.rehearsal.note}
                          </Text>
                        </RenderWhen>
                        <RenderWhen condition={found.rehearsal.error}>
                          <Alert tone="error" title={found.rehearsal.error} />
                        </RenderWhen>
                        <RenderWhen condition={found.rehearsal.schema_diff}>
                          <div>
                            <Eyebrow className="mb-1.5 block">schema diff</Eyebrow>
                            <SchemaDiff diff={found.rehearsal.schema_diff ?? ''} />
                          </div>
                        </RenderWhen>
                        <RenderWhen condition={found.rehearsal.row_diff}>
                          <div>
                            <Eyebrow className="mb-1.5 block">row diff</Eyebrow>
                            <div className="overflow-hidden rounded-lg border border-line">
                              <SqlEditor
                                value={found.rehearsal.row_diff ?? ''}
                                height={140}
                                readOnly
                              />
                            </div>
                          </div>
                        </RenderWhen>
                      </Card>
                    ) : null}
                  </RenderWhen>
                </div>

                <aside className="flex flex-col gap-4 lg:sticky lg:top-6">
                  <Card
                    title="Decision"
                    loading={actions.decide.isPending}
                    bodyClassName="flex flex-col gap-2 p-4"
                    footer={
                      <>
                        <div className="flex w-full flex-wrap items-center gap-2">
                          <AppButton
                            intent="primary"
                            icon={<CheckOutlined />}
                            disabled={
                              !can['review:proposal'] ||
                              ['approved', 'applied'].includes(found.status)
                            }
                            loading={busy}
                            onClick={() =>
                              void run(
                                () => actions.decide.mutateAsync({ approve: true }),
                                'approved',
                              )
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
                            onClick={() =>
                              void run(() => actions.rehearse.mutateAsync(), 'rehearsed')
                            }
                          >
                            Rehearse again
                          </AppButton>
                        </div>

                        <Can
                          do="execute:proposal"
                          fallback={
                            <Text size="xs" tone="muted">
                              apply needs the apply scope
                            </Text>
                          }
                        >
                          <AppButton
                            intent="primary"
                            block
                            icon={<ThunderboltOutlined />}
                            disabled={!can['execute:proposal'] || found.status !== 'approved'}
                            onClick={apply}
                          >
                            Apply
                          </AppButton>
                        </Can>
                      </>
                    }
                  >
                    <RenderWhen condition={confirming}>
                      <Text size="xs" tone="muted">
                        rejecting…
                      </Text>
                    </RenderWhen>

                    <RenderWhen
                      condition={found.decided_by}
                      fallback={
                        <Text as="p" size="sm" tone="muted">
                          Waiting for a reviewer.
                        </Text>
                      }
                    >
                      <div className="flex flex-col gap-0.5">
                        <Eyebrow>decided</Eyebrow>
                        <Text size="sm">
                          {found.decided_by} · {formatWhen(found.decided_at)}
                        </Text>
                        <RenderWhen condition={found.decided_note}>
                          <Text as="p" size="xs" tone="muted">
                            {found.decided_note}
                          </Text>
                        </RenderWhen>
                      </div>
                    </RenderWhen>

                    <RenderWhen condition={found.applied_by}>
                      <div className="flex flex-col gap-0.5">
                        <Eyebrow>applied</Eyebrow>
                        <Text size="sm">
                          {found.applied_by} · {formatWhen(found.applied_at)}
                        </Text>
                      </div>
                    </RenderWhen>

                    <RenderWhen condition={found.progress_stage}>
                      <div className="flex flex-col gap-0.5">
                        <Eyebrow>running</Eyebrow>
                        <Text size="sm">
                          {found.progress_stage}
                          {found.progress_percent !== undefined && found.progress_percent >= 0
                            ? ` · ${found.progress_percent}%`
                            : ''}
                        </Text>
                      </div>
                    </RenderWhen>

                    <Can
                      do="review:proposal"
                      fallback={
                        <Text size="xs" tone="muted">
                          {identity?.name} cannot review this change
                        </Text>
                      }
                    >
                      <Text size="xs" tone="muted">
                        approve and reject are journaled against your token
                      </Text>
                    </Can>
                  </Card>

                  <Card
                    title="History"
                    loading={events.isLoading}
                    bodyClassName="p-4"
                    extra={`${(events.data ?? []).length} entries`}
                  >
                    <RenderWhen
                      condition={(events.data ?? []).length > 0}
                      fallback={
                        <Text size="sm" tone="muted">
                          Nothing recorded yet.
                        </Text>
                      }
                    >
                      <Timeline
                        items={(events.data ?? []).map((entry) => ({
                          color: 'var(--color-brand)',
                          children: (
                            <div className="flex flex-col gap-0.5">
                              <Text size="sm" className="font-medium capitalize">
                                {entry.kind}
                              </Text>
                              <Text size="xs" tone="muted">
                                {entry.actor} · {formatWhen(entry.created_at)}
                              </Text>
                              {Object.keys(entry.detail ?? {}).length > 0 ? (
                                <Text size="xs" tone="muted" mono>
                                  {Object.entries(entry.detail)
                                    .map(([key, value]) => `${key}=${value}`)
                                    .join('  ')}
                                </Text>
                              ) : null}
                            </div>
                          ),
                        }))}
                      />
                    </RenderWhen>
                  </Card>
                </aside>
              </div>
            </div>
          ) : null}
        </RenderWhen>
      </AsyncState>
    </section>
  );
}
