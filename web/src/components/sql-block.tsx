import { Tag } from 'antd';
import { clsx } from 'clsx';
import { useState } from 'react';
import { AppButton } from './app-button';

// A statement is shown as a statement: a quiet block of SQL with a copy action, not a `<code>` tag in
// the middle of a paragraph. The colour is the only dark surface in the console, because code that
// looks like prose is code nobody reads carefully.
export function SqlBlock({
  sql,
  label,
  copyable = true,
}: {
  sql: string;
  label?: string;
  copyable?: boolean;
}) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    await navigator.clipboard.writeText(sql);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  return (
    <figure className="m-0">
      <div className="mb-2 flex items-center gap-2">
        <figcaption className="text-xs font-medium tracking-wide text-muted uppercase">
          {label ?? 'statement'}
        </figcaption>
        <span className="flex-1" />
        {copyable ? (
          <AppButton
            size="small"
            intent="quiet"
            onClick={() => void copy()}
            aria-live="polite"
            aria-label={copied ? 'copied' : 'copy the statement'}
          >
            {copied ? 'Copied' : 'Copy'}
          </AppButton>
        ) : null}
      </div>
      <pre
        className={clsx(
          'max-h-72 overflow-auto rounded-lg bg-[#0f172a] px-3.5 py-3 font-mono text-[12.5px]',
          'leading-relaxed whitespace-pre-wrap break-words text-[#e2e8f0]',
        )}
      >
        {sql}
      </pre>
    </figure>
  );
}

// A status is a tag, and the colour is decided here so two screens cannot colour "failed" differently.
const COLOURS: Record<string, string> = {
  verified: 'green',
  partial: 'gold',
  failed: 'red',
  approved: 'blue',
  applied: 'green',
  applying: 'processing',
  rejected: 'default',
  archived: 'default',
};

export function StatusTag({ status }: { status: string }) {
  return (
    <Tag color={COLOURS[status] ?? 'default'} className="capitalize">
      {status}
    </Tag>
  );
}
