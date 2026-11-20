import type { ThemeConfig } from 'antd';

// One orange, one white, one grey scale — and nothing else. Every colour in the console comes from
// this file, so a screen cannot invent a second orange: `grep -r '#[0-9a-fA-F]\{6\}' src` finds this
// file and the two gradients below, and nowhere else.
export const orange = '#f97316';
export const orangeSoft = '#fff7ed';
export const ink = '#1f2937';
export const muted = '#6b7280';
export const line = '#e5e7eb';
export const surface = '#ffffff';

export const theme: ThemeConfig = {
  token: {
    colorPrimary: orange,
    colorInfo: orange,
    colorLink: orange,
    colorTextBase: ink,
    colorBgBase: surface,
    colorBorder: line,
    colorTextSecondary: muted,
    borderRadius: 8,
    controlHeight: 36,
    fontFamily:
      "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
    fontSize: 14,
    wireframe: false,
  },
  components: {
    Layout: { headerBg: surface, siderBg: surface, bodyBg: '#fafafa' },
    Menu: { itemSelectedColor: orange, itemActiveBg: orangeSoft },
    Button: { primaryShadow: 'none' },
    Card: { headerBg: surface },
    Tabs: { inkBarColor: orange, itemSelectedColor: orange, itemHoverColor: orange },
    Table: { headerBg: '#fafafa', rowHoverBg: orangeSoft },
    Tag: { defaultBg: orangeSoft, defaultColor: orange },
  },
};
