import { Alert, Skeleton } from 'antd';
import type { ReactNode } from 'react';
import { EmptyState } from './empty-state';
import { RenderWhen } from './render-when';

// One boundary for every query: loading, failed, empty, ready — in that order, and in one place. A
// screen passes the state a hook returned and never renders a skeleton itself.
export function AsyncState({
  loading,
  error,
  empty,
  emptyDescription = 'Nothing here yet',
  emptyAction,
  onRetry,
  children,
}: {
  loading: boolean;
  error?: unknown;
  empty?: boolean;
  emptyDescription?: string;
  emptyAction?: ReactNode;
  onRetry?: () => void;
  children: ReactNode;
}) {
  if (loading) {
    return (
      <div aria-busy="true" aria-live="polite" className="flex flex-col gap-3">
        <Skeleton active paragraph={{ rows: 2 }} />
        <Skeleton active paragraph={{ rows: 4 }} />
      </div>
    );
  }

  if (error) {
    return (
      <Alert
        type="error"
        showIcon
        message="That did not work"
        description={error instanceof Error ? error.message : String(error)}
        action={
          onRetry ? (
            <button type="button" onClick={onRetry} className="px-2 text-brand underline">
              Try again
            </button>
          ) : null
        }
      />
    );
  }

  return (
    <RenderWhen
      condition={!empty}
      fallback={<EmptyState description={emptyDescription} action={emptyAction} />}
    >
      {children}
    </RenderWhen>
  );
}
