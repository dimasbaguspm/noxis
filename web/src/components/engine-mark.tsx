import { Tooltip } from 'antd';

// The engine, as a mark rather than a word. Shipping somebody else's trademark means shipping their
// artwork faithfully, and a hand-drawn dolphin is worse than no dolphin: what a row needs is "database,
// and which one", at a glance, in this palette. The name survives where it has to — the tooltip for a
// pointer, the aria-label for a screen reader.

type Engine = 'mysql' | 'mariadb' | string;

const LOOK: Record<'mysql' | 'mariadb', { name: string; tint: string; soft: string }> = {
  mysql: { name: 'MySQL', tint: 'var(--color-brand)', soft: 'var(--color-brand-soft)' },
  mariadb: { name: 'MariaDB', tint: 'var(--color-ink)', soft: 'var(--color-tint)' },
};

export function EngineMark({ engine, size = 14 }: { engine: Engine; size?: number }) {
  const look = LOOK[engine === 'mariadb' ? 'mariadb' : 'mysql'];
  return (
    <Tooltip title={look.name}>
      <span
        role="img"
        aria-label={look.name}
        className="inline-flex items-center justify-center rounded-md"
        style={{ background: look.soft, width: size + 10, height: size + 10 }}
      >
        <svg width={size} height={size} viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <ellipse cx="12" cy="6" rx="7" ry="3" stroke={look.tint} strokeWidth="1.7" />
          <path d="M5 6v6c0 1.66 3.13 3 7 3s7-1.34 7-3V6" stroke={look.tint} strokeWidth="1.7" />
          <path d="M5 12v6c0 1.66 3.13 3 7 3s7-1.34 7-3v-6" stroke={look.tint} strokeWidth="1.7" />
        </svg>
      </span>
    </Tooltip>
  );
}
