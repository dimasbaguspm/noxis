// The mark, in the console's own palette. The drawing is the one the owner supplied — two strokes that
// read as an N — recoloured into the brand gradient and given a top sheen and a hairline edge, because
// a flat square at 24px reads as a redaction rather than a logo.
//
// `tone="onBrand"` is the same mark reversed, for the orange panel where a second orange would vanish.

export function Logo({
  size = 28,
  tone = 'brand',
  className,
}: {
  size?: number;
  tone?: 'brand' | 'onBrand';
  className?: string;
}) {
  const reversed = tone === 'onBrand';
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 512 512"
      role="img"
      aria-label="noXis"
      className={className}
    >
      <defs>
        <linearGradient id="noxis-logo-fill" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="var(--color-brand-light)" />
          <stop offset="100%" stopColor="var(--color-brand-dark)" />
        </linearGradient>
        <linearGradient id="noxis-logo-sheen" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="var(--color-surface)" stopOpacity="0.3" />
          <stop offset="58%" stopColor="var(--color-surface)" stopOpacity="0" />
        </linearGradient>
      </defs>

      <rect
        width="512"
        height="512"
        rx="112"
        fill={reversed ? 'var(--color-surface)' : 'url(#noxis-logo-fill)'}
      />
      <rect width="512" height="512" rx="112" fill="url(#noxis-logo-sheen)" />
      <rect
        x="4"
        y="4"
        width="504"
        height="504"
        rx="108"
        fill="none"
        stroke="var(--color-surface)"
        strokeOpacity="0.22"
        strokeWidth="6"
      />
      <path
        d="M790 66L79 1246L337 1403L931 418L900 572L878 752L1000 966L1187 780L1238 335L792 66Z M1674 598L1080 1586L1113 1415L1132 1249L1023 1034L826 1211L775 1666L1222 1936L1934 755L1675 598Z"
        fill={reversed ? 'var(--color-brand)' : 'var(--color-surface)'}
        transform="translate(256 256) scale(0.20809) translate(-1006.5 -1001)"
      />
    </svg>
  );
}
