import type { ThemeConfig } from 'antd';

// The same values as `@theme` in styles.css, and the reason that duplication is acceptable: Ant Design
// takes numbers and hex strings at runtime, Tailwind takes CSS variables at build time, and neither
// reads the other. Both files name the pair, and nothing else in the console writes a colour.
export const brand = '#f97316';
export const brandSoft = '#fff7ed';
export const ink = '#1f2937';
export const muted = '#6b7280';
export const line = '#e5e7eb';

export const theme: ThemeConfig = {
  token: {
    colorPrimary: brand,
    colorInfo: brand,
    colorLink: brand,
    colorTextBase: ink,
    colorBgBase: '#ffffff',
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
    Layout: { headerBg: '#ffffff', bodyBg: '#fafafa' },
    Menu: { itemSelectedColor: brand, itemActiveBg: brandSoft },
    Button: { primaryShadow: 'none' },
    Tabs: { inkBarColor: brand, itemSelectedColor: brand, itemHoverColor: brand },
    Table: { headerBg: '#fafafa', rowHoverBg: brandSoft },
    Tag: { defaultBg: brandSoft, defaultColor: brand },
    Segmented: { itemSelectedBg: '#ffffff', itemSelectedColor: brand },
  },
};
