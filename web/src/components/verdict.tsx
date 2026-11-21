import type { ProposalDetail } from '../api/types';
import { Text } from './text';

// The proposal page used to show everything and say nothing: a reviewer scrolled three cards of SQL and
// diffs before learning whether the rehearsal had proved the change, and the buttons that decide were
// buried in the footer of the first card. This module is the sentence that comes first, and the facts
// behind it — derived from the proposal, never typed per status, so a new status cannot be silent.

export type VerdictTone = 'ok' | 'warning' | 'failure' | 'pending';

export type VerdictFact = { label: string; value: string };

export type Verdict = {
  tone: VerdictTone;
  sentence: string;
  facts: VerdictFact[];
};

// What a tier actually proves, in the engine's own terms.
const TIER_CLAIM: Record<string, string> = {
  static: 'parsed and explained, nothing was run',
  shadow: 'rebuilt in a throwaway database, no rows copied',
  rehearsal: 'rows copied and rolled back',
};

const STATUS_WORD: Record<string, string> = {
  verified: 'rehearsed',
  partial: 'partly rehearsed',
  failed: 'not proven',
  unsupported: 'cannot be rehearsed here',
};

/** firstLine keeps an engine error readable in one sentence: the rest is in the card below. */
function firstLine(text: string | undefined): string {
  return (text ?? '').split('\n')[0]?.trim() ?? '';
}

/**
 * verdictOf is the page's one sentence. What it may claim is bounded by what the tier actually ran: a
 * static check reads a plan and never runs the rollback, so saying the rollback was proven there would
 * be a lie in the one place a reviewer trusts most.
 */
export function verdictOf(proposal: ProposalDetail): Verdict {
  const rehearsal = proposal.rehearsal;
  const took = rehearsal?.duration_ms !== undefined ? ` in ${rehearsal.duration_ms} ms` : '';

  const facts: VerdictFact[] = [{ label: 'proof', value: tierClaim(proposal.tier) }];
  if (rehearsal?.duration_ms !== undefined) {
    facts.push({ label: 'took', value: `${rehearsal.duration_ms} ms` });
  }
  if (proposal.rows_affected) {
    facts.push({ label: 'rows', value: String(proposal.rows_affected) });
  }
  if (proposal.algorithm) {
    facts.push({ label: 'algorithm', value: proposal.algorithm });
  }
  facts.push({ label: 'rollback', value: rollbackClaim(proposal.rollback, rehearsal?.status) });

  if (!rehearsal) {
    return {
      tone: 'pending',
      sentence: 'Not rehearsed yet: nothing here has been proven against a database.',
      facts,
    };
  }

  if (rehearsal.status === 'failed' || rehearsal.status === 'unsupported') {
    const why = firstLine(rehearsal.error) || firstLine(rehearsal.note);
    const word = STATUS_WORD[rehearsal.status] ?? 'did not prove this';
    return {
      tone: 'failure',
      sentence: `The rehearsal was ${word}${why ? `: ${why}` : '.'}`,
      facts,
    };
  }

  const rows = proposal.rows_affected;
  const algorithm = proposal.algorithm;

  if (proposal.status === 'applied' && rows !== undefined && algorithm) {
    return {
      tone: proposal.irreversible ? 'warning' : 'ok',
      sentence: `Applied: ${rows} row${rows === 1 ? '' : 's'} changed with ${algorithm}${
        proposal.irreversible
          ? ', and there is no way back.'
          : ', and the rollback restores what it changed.'
      }`,
      facts,
    };
  }
  if (proposal.status === 'applied') {
    return {
      tone: proposal.irreversible ? 'warning' : 'ok',
      sentence: `Applied by ${proposal.applied_by ?? 'someone'}${
        proposal.irreversible ? ', and there is no way back.' : '.'
      }`,
      facts,
    };
  }
  if (proposal.irreversible) {
    return {
      tone: 'warning',
      sentence:
        'Rehearsed, but nothing can undo it: the change carries no rollback, so this is the one decision without a way back.',
      facts,
    };
  }
  if (proposal.tier === 'static') {
    return {
      tone: 'warning',
      sentence: `Checked statically${took}: the statement was parsed and its plan read, nothing was run, so the rollback is unproven.`,
      facts,
    };
  }
  if (proposal.tier === 'rehearsal') {
    return {
      tone: rehearsal.status === 'verified' ? 'ok' : 'warning',
      sentence: `Ran against copied rows in a throwaway database${took}, and the rollback undid what it changed.`,
      facts,
    };
  }
  return {
    tone: rehearsal.status === 'verified' ? 'ok' : 'warning',
    sentence: `Rebuilt in a throwaway database${took} and the change ran there; the rollback put the schema back.`,
    facts,
  };
}

function tierClaim(tier: string): string {
  const claim = TIER_CLAIM[tier];
  return claim ? `${tier} · ${claim}` : tier;
}

function rollbackClaim(rollback: string | undefined, rehearsalStatus: string | undefined): string {
  if (!rollback) return 'none';
  // "proven" is a claim about the rehearsal, not about the card: only a verified run makes it.
  return rehearsalStatus === 'verified' ? 'given, and proven' : 'given, unproven';
}

const TONE_CLASS: Record<VerdictTone, string> = {
  ok: 'border-green-200 bg-green-50 text-green-900',
  warning: 'border-amber-200 bg-amber-50 text-amber-900',
  failure: 'border-red-200 bg-red-50 text-red-900',
  pending: 'border-line bg-surface text-ink',
};

/**
 * Verdict is the first thing a reviewer reads: one sentence that says what was proven, and the facts
 * that sentence rests on. It is not an Alert — nothing is wrong — so it is a band with a tone, not a
 * banner with an icon. Two lines at most: the proof itself is the card below, and a banner that grows
 * into a second card pushes the statement off the first screen.
 */
export function Verdict({ verdict }: { verdict: Verdict }) {
  return (
    <div
      className={`flex flex-wrap items-baseline gap-x-3 gap-y-1 rounded-xl border px-3.5 py-2 ${TONE_CLASS[verdict.tone]}`}
    >
      <Text as="p" size="sm" className="line-clamp-2 min-w-0 flex-1 font-medium">
        {verdict.sentence}
      </Text>
      <Text size="xs" className="max-w-full truncate opacity-70">
        {verdict.facts.map((fact) => `${fact.label} ${fact.value}`).join(' · ')}
      </Text>
    </div>
  );
}
