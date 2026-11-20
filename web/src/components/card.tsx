import { Card as AntCard } from 'antd';
import { clsx } from 'clsx';
import type { ReactNode } from 'react';
import { Text } from './text';

// Ant Design's Card is the right frame — a head, a body, a foot, one border — with a head sized for
// marketing pages and no footer at all. Wrapped, so the console has one card: a small title, an
// optional right-hand fact, and a footer where the action lives. The proportions are in `.noxis-card`
// in styles.css.

export function Card({
  title,
  extra,
  footer,
  children,
  className,
  bodyClassName,
  loading = false,
}: {
  title?: ReactNode;
  extra?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
  loading?: boolean;
}) {
  return (
    <AntCard
      className={clsx('noxis-card', className)}
      loading={loading}
      title={title ? <span className="text-sm font-semibold text-ink">{title}</span> : undefined}
      extra={
        extra ? (
          <Text size="xs" tone="muted">
            {extra}
          </Text>
        ) : undefined
      }
      styles={{ body: { padding: 0 } }}
    >
      <div className={bodyClassName}>{children}</div>
      {footer ? (
        <div className="noxis-card-footer flex flex-wrap items-center gap-2">{footer}</div>
      ) : null}
    </AntCard>
  );
}
