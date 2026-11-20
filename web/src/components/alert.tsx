import { Alert as AntAlert } from 'antd';
import type { ReactNode } from 'react';
import { clsx } from 'clsx';

// Ant Design's Alert is the right behaviour — a live region, an icon, an optional action — with the
// wrong proportions for this console: it was built for dense admin screens, and a message that a
// person has to read twice is a message rendered too small. The wrapper fixes the size and the
// spacing once, so a page cannot invent a fifth kind of banner.

export type Tone = 'info' | 'success' | 'warning' | 'error';

export function Alert({
  tone = 'info',
  title,
  description,
  action,
  className,
}: {
  tone?: Tone;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <AntAlert
      className={clsx('noxis-alert', `noxis-alert-${tone}`, className)}
      type={tone}
      showIcon
      message={title}
      description={description}
      action={action}
    />
  );
}
