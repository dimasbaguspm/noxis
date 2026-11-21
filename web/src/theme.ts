import type { ThemeConfig } from 'antd';

// The same values as `@theme` in styles.css, and the reason that duplication is acceptable: Ant Design
// takes numbers and hex strings at runtime, Tailwind takes CSS variables at build time, and neither
// reads the other. Both files name the pair, and nothing else in the console writes a colour.
//
// Two oranges, on purpose. `brand` is the identity — the logo ramp, focus marks, the separator that
// carries no words. `brandInk` is the same hue, dark enough to carry text: white on `brand` is 2.8:1 and
// fails WCAG AA, while `brandInk` is 5.18:1 in both directions. A primary button or a link is text.
export const brand = '#f97316';
export const brandSoft = '#fff7ed';
export const brandLight = '#fb923c';
export const brandDark = '#ea580c';
export const brandInk = '#c2410c';
export const ink = '#1f2937';
export const muted = '#6b7280';
export const line = '#e5e7eb';
export const surface = '#ffffff';
export const tint = '#f3f4f6';
export const codeBg = '#0f172a';
export const codeInk = '#e2e8f0';

export const theme: ThemeConfig = {
  token: {
    colorPrimary: brandInk,
    colorInfo: brandInk,
    colorLink: brandInk,
    colorTextBase: ink,
    colorBgBase: surface,
    colorBorder: line,
    colorTextSecondary: muted,
    borderRadius: 8,
    controlHeight: 36,
    fontSize: 14,
    fontFamily:
      "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    wireframe: false,
  },
  components: {
    Layout: { headerBg: surface, bodyBg: '#fafafa' },
    Menu: { itemSelectedColor: brandInk, itemActiveBg: brandSoft },
    Button: { primaryShadow: 'none' },
    Tabs: { inkBarColor: brandInk, itemSelectedColor: brandInk, itemHoverColor: brandInk },
    Table: { headerBg: '#fafafa', rowHoverBg: brandSoft },
    Tag: { defaultBg: brandSoft, defaultColor: brandInk },
    Segmented: { itemSelectedBg: surface, itemSelectedColor: brandInk },
  },
};
