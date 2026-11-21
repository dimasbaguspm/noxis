import { CheckOutlined, CloseOutlined, RedoOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { App as AntApp, Flex, Input, Tag, Timeline } from 'antd';
import { useState } from 'react';
import { useParams } from 'react-router-dom';
import { useSession } from '../api/session-context';
import { usePermissions } from '../api/permissions';
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
import { TextDiff } from '../editor/diff-editor';
import { SqlEditor } from '../editor/sql-editor';
import { formatWhen } from '../lib/when';

// One proposal, with its evidence, under the instance it belongs to. The console decides nothing: it
// sends the same requests the CLI does and shows what the server answered.
//
// The page reads in the order a reviewer actually works: what was proven (the verdict), what it is (the
// statement), what undoes it (the rollback), how it was proven (the diffs), and only then the decision —
// which sits in the aside so the buttons are where the reading ends, not where it began.

type Decision = 'approve' | 'reject' | 'rehearse' | 'apply';

// What each status still allows, copied from the domain's own transition table
// (`internal/domain/proposal.go`). A button the server would refuse is a button that lies, so `applied`
// allows nothing and `verified` cannot apply what nobody approved.
const ALLOWED: Record<string, Decision[]> = {
  draft: ['rehearse'],
  failed: ['rehearse'],
  verified: ['approve', 'reject', 'rehearse'],
  approved: ['apply', 'reject', 'rehearse'],
  applying: [],
  applied: [],
  rejected: [],
};

// The one line that says what is left to do, in the same order the history below records it.
const NEXT_STEP: Record<string, string> = {
  draft: 'Next: rehearse it — nothing has been proven yet.',
  failed: 'Next: rehearse it again; the last run did not prove it.',
  verified: 'Next: approve or reject it.',
  approved: 'Next: apply it, or reject it before it lands.',
  applying: 'Running: the apply is in progress.',
  applied: 'Done: this change was applied.',
  rejected: 'Closed: it was rejected and never applied.',
};

// The journal's own vocabulary, said the way a person reads it, and a colour per kind: the same green
// for "applied" here and in a status tag, so one glance means one thing.
const EVENT_WORD: Record<string, string> = {
  proposed: 'Proposed',
  rehearsed: 'Rehearsed',
  approved: 'Approved',
  rejected: 'Rejected',
  applied: 'Applied',
  apply_failed: 'Apply failed',
  override: 'Override',
};

const EVENT_COLOUR: Record<string, string> = {
  proposed: 'var(--color-brand)',
  rehearsed: 'var(--color-brand)',
  approved: '#1677ff',
  rejected: 'var(--color-muted)',
  applied: '#389e0d',
  apply_failed: '#ff4d4f',
  override: '#faad14',
};

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
  // A button is live only when the status still allows it: the same list the server enforces.
  const permits = (decision: Decision) => (ALLOWED[found?.status ?? ''] ?? []).includes(decision);
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
                  {/* The rollback belongs to the change, not beside it: an operator copies both and runs one
                      after the other, so one card carries the statement, the way back, and the note. */}
                  <Card
                    title="Statement"
                    extra={`proposed by ${found.proposed_by} · ${formatWhen(found.created_at)}`}
                    bodyClassName="flex flex-col gap-3 p-4"
                  >
                    <div>
                      <Eyebrow className="mb-1.5 block">change</Eyebrow>
                      <div className="overflow-hidden rounded-lg border border-line">
                        <SqlEditor value={found.query} height={140} readOnly />
                      </div>
                    </div>

                    <div>
                      <Eyebrow className="mb-1.5 block">rollback</Eyebrow>
                      {found.rollback ? (
                        <div className="overflow-hidden rounded-lg border border-line">
                          <SqlEditor value={found.rollback} height={120} readOnly />
                        </div>
                      ) : (
                        <Alert tone="warning" title="No rollback: this change is irreversible" />
                      )}
                    </div>

                    <RenderWhen condition={found.note}>
                      <Text as="p" size="sm" tone="muted">
                        {found.note}
                      </Text>
                    </RenderWhen>
                  </Card>

                  <RenderWhen condition={found.rehearsal}>
                    {found.rehearsal ? (
                      // The verdict above already says what was proven. This card is the proof itself, so it
                      // carries only what the verdict does not: the tags, the note, and the diffs — with
                      // "nothing changed" said in words instead of left as an empty block.
                      <Card
                        title="Evidence"
                        extra={`${found.rehearsal.tier} · ${found.rehearsal.duration_ms ?? 0} ms`}
                        bodyClassName="flex flex-col gap-3 p-4"
                      >
                        <div className="flex flex-wrap items-center gap-2">
                          <StatusTag status={found.rehearsal.status} />
                          <RenderWhen condition={found.rehearsal.irreversible}>
                            <Tag color="red" className="m-0">
                              irreversible
                            </Tag>
                          </RenderWhen>
                          <Text size="xs" tone="muted">
                            {formatWhen(found.rehearsal.created_at)}
                          </Text>
                        </div>
                        <RenderWhen condition={found.rehearsal.note}>
                          <Text as="p" size="sm" tone="muted">
                            {found.rehearsal.note}
                          </Text>
                        </RenderWhen>
                        <RenderWhen condition={found.rehearsal.error}>
                          <Alert tone="error" title={found.rehearsal.error} />
                        </RenderWhen>
                        <Proof
                          label="schema"
                          diff={found.rehearsal.schema_diff}
                          none="the schema did not change"
                        />
                        <Proof
                          label="rows"
                          diff={found.rehearsal.row_diff}
                          none="no rows were touched"
                        />
                      </Card>
                    ) : null}
                  </RenderWhen>
                </div>

                {/* The decision and the history are one column, in the order they happen: what is left to
                    do on top, what was already done under it. */}
                <aside className="flex flex-col gap-4 lg:sticky lg:top-6">
                  <Card
                    title="Decision"
                    bodyClassName="flex flex-col gap-2.5 p-4"
                    footer={
                      <>
                        <div className="grid w-full grid-cols-2 gap-2">
                          <AppButton
                            intent="primary"
                            block
                            icon={<CheckOutlined />}
                            disabled={!can['review:proposal'] || !permits('approve')}
                            loading={actions.decide.isPending && confirming === null}
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
                            block
                            icon={<CloseOutlined />}
                            disabled={!can['review:proposal'] || !permits('reject')}
                            loading={confirming !== null}
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
                        </div>
                        <AppButton
                          intent="primary"
                          block
                          icon={<ThunderboltOutlined />}
                          disabled={!can['execute:proposal'] || !permits('apply')}
                          loading={actions.apply.isPending}
                          onClick={apply}
                        >
                          Apply
                        </AppButton>
                        <AppButton
                          block
                          icon={<RedoOutlined />}
                          disabled={!permits('rehearse')}
                          loading={actions.rehearse.isPending}
                          onClick={() =>
                            void run(() => actions.rehearse.mutateAsync(), 'rehearsed')
                          }
                        >
                          Rehearse again
                        </AppButton>
                        <Text size="xs" tone="muted">
                          {NEXT_STEP[found.status] ?? ''}
                        </Text>
                      </>
                    }
                  >
                    <RenderWhen
                      condition={found.decided_by}
                      fallback={
                        <Text as="p" size="sm" tone="muted">
                          {permits('approve') ? 'Waiting for a reviewer.' : 'No decision recorded.'}
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

                    <RenderWhen
                      condition={
                        !can['review:proposal'] || !can['execute:proposal'] || !can['propose']
                      }
                    >
                      <Text size="xs" tone="muted">
                        {!can['review:proposal'] && !can['execute:proposal']
                          ? `${identity?.name} can read this change but not decide it`
                          : !can['execute:proposal']
                            ? 'applying needs the apply scope'
                            : 'rehearsing needs the propose scope'}
                      </Text>
                    </RenderWhen>
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
                          color: EVENT_COLOUR[entry.kind] ?? 'var(--color-line)',
                          children: (
                            <div className="flex min-w-0 flex-col gap-0.5">
                              <Text size="sm" className="font-medium">
                                {EVENT_WORD[entry.kind] ?? entry.kind}
                              </Text>
                              <Text size="xs" tone="muted">
                                {formatWhen(entry.created_at)} · {entry.actor}
                              </Text>
                              {Object.entries(entry.detail ?? {}).map(([key, value]) => (
                                <Text key={key} size="xs" tone="muted" mono className="truncate">
                                  {key} {value}
                                </Text>
                              ))}
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

/** One diff, or the sentence that replaces it: an empty block reads as a failed load, not as a fact. */
function Proof({ label, diff, none }: { label: string; diff?: string; none: string }) {
  return (
    <div className="flex flex-col gap-1">
      <Eyebrow>{label} diff</Eyebrow>
      {diff ? (
        <TextDiff diff={diff} />
      ) : (
        <Text size="sm" tone="muted">
          {none}
        </Text>
      )}
    </div>
  );
}
