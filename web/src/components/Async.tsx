import { useState } from 'react';
import type { ReactNode } from 'react';
import { Alert, Button, Flex, Skeleton } from 'antd';

// Every screen answers the same three questions before it shows anything: is it loading, did it come
// back empty, did it fail. One component, so a screen cannot forget one of them.

export type AsyncState<T> =
  | { kind: 'loading' }
  | { kind: 'failed'; error: string }
  | { kind: 'ready'; value: T };

// useLoader is the smallest thing that turns a promise into those three states: no cache, no retry
// policy, nothing the console does not use.
export function useLoader<T>(): [AsyncState<T>, (load: () => Promise<T>) => Promise<void>] {
  const [state, setState] = useState<AsyncState<T>>({ kind: 'loading' });
  const run = async (load: () => Promise<T>) => {
    setState({ kind: 'loading' });
    try {
      setState({ kind: 'ready', value: await load() });
    } catch (cause) {
      setState({ kind: 'failed', error: cause instanceof Error ? cause.message : String(cause) });
    }
  };
  return [state, run];
}

export function Async<T>({
  state,
  empty,
  isEmpty,
  children,
  retry,
}: {
  state: AsyncState<T>;
  empty?: ReactNode;
  isEmpty?: (value: T) => boolean;
  children: (value: T) => ReactNode;
  retry?: () => void;
}) {
  if (state.kind === 'loading') {
    return (
      <Flex vertical gap={12} aria-busy="true" aria-live="polite">
        <Skeleton active paragraph={{ rows: 2 }} />
        <Skeleton active paragraph={{ rows: 4 }} />
      </Flex>
    );
  }
  if (state.kind === 'failed') {
    return (
      <Alert
        type="error"
        showIcon
        message="That did not work"
        description={state.error}
        action={
          retry ? (
            <Button size="small" onClick={retry}>
              Try again
            </Button>
          ) : null
        }
      />
    );
  }
  if (isEmpty?.(state.value) && empty) return <>{empty}</>;
  return <>{children(state.value)}</>;
}
