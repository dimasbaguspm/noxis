// LogoMark is the noXis brand mark, inline so it takes the surrounding text
// colour — the sidebar renders it on the dark rail, the README uses the file.
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg viewBox="48 42 1918 1918" className={className} aria-hidden="true" focusable="false">
      <path
        d="M790 66L79 1246L337 1403L931 418L900 572L878 752L1000 966L1187 780L1238 335L792 66Z M1674 598L1080 1586L1113 1415L1132 1249L1023 1034L826 1211L775 1666L1222 1936L1934 755L1675 598Z"
        fill="currentColor"
      />
    </svg>
  );
}
