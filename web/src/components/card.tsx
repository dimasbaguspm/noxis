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
  stretch = false,
}: {
  title?: ReactNode;
  extra?: ReactNode;
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
  loading?: boolean;
  /** Reaches the bottom of its cell, body included: for cards that sit beside a taller one. */
  stretch?: boolean;
}) {
  return (
    <AntCard
      className={clsx('noxis-card', stretch && 'noxis-card-stretch', className)}
      loading={loading}
      title={title ? <Text className="font-semibold">{title}</Text> : undefined}
      extra={
        extra ? (
          <Text size="xs" tone="muted">
            {extra}
          </Text>
        ) : undefined
      }
      styles={{ body: { padding: 0 } }}
    >
      <div className={clsx(stretch && 'flex flex-1 flex-col', bodyClassName ?? 'p-4')}>
        {children}
      </div>
      {footer ? (
        <div className="noxis-card-footer flex flex-wrap items-center gap-2">{footer}</div>
      ) : null}
    </AntCard>
  );
}
