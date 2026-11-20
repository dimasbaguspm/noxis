import { Empty } from 'antd';
import type { ReactNode } from 'react';

// Nothing to show is a state, not an absence: one component, with a sentence and the action that
// would change it.
export function EmptyState({
  description,
  action,
  compact,
}: {
  description: string;
  action?: ReactNode;
  compact?: boolean;
}) {
  return (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={description}
      className={compact ? 'my-4' : 'my-10'}
    >
      {action}
    </Empty>
  );
}
