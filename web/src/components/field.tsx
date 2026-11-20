import { Typography } from 'antd';
import { clsx } from 'clsx';
import type { ReactNode } from 'react';

// The label, the control, and one line under it. Every field component renders this, so the spacing
// and the error position are the same everywhere and no page positions a message by hand.
export function Field({
  label,
  htmlFor,
  error,
  hint,
  fullWidth,
  children,
}: {
  label: string;
  htmlFor: string;
  error?: string;
  hint?: string;
  fullWidth?: boolean;
  children: ReactNode;
}) {
  const message = error ?? hint;
  return (
    <div className={clsx('mb-4', fullWidth && 'w-full')}>
      <label htmlFor={htmlFor} className="mb-1.5 block text-[13px] font-medium text-ink">
        {label}
      </label>
      {children}
      {message ? (
        <Typography.Text
          id={`${htmlFor}-message`}
          type={error ? 'danger' : 'secondary'}
          className="mt-1.5 block text-xs"
        >
          {message}
        </Typography.Text>
      ) : null}
    </div>
  );
}
