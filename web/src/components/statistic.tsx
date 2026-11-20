import { Statistic as AntStatistic } from 'antd';
import type { ReactNode } from 'react';
import { Eyebrow } from './text';

// Ant Design's Statistic is the right big number — right aligned digits, a suffix, a formatter — with
// a title style this console does not use. Wrapped, so a screen names a label and a value and the
// proportions are decided here: 22px, semibold, ink, with a quiet hint underneath.

export function Statistic({
  label,
  value,
  suffix,
  hint,
}: {
  label: string;
  value: ReactNode;
  suffix?: string;
  hint?: ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <Eyebrow>{label}</Eyebrow>
      <AntStatistic
        value={value as string | number}
        groupSeparator=""
        suffix={suffix}
        valueStyle={{
          fontSize: 22,
          fontWeight: 600,
          color: 'var(--color-ink)',
          lineHeight: 1.25,
          letterSpacing: '-0.01em',
        }}
      />
      {hint ? <span className="truncate text-xs text-muted">{hint}</span> : null}
    </div>
  );
}
