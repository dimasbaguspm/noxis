import { Avatar, Dropdown, Flex, Skeleton, Tag, Tooltip } from 'antd';
import { LogoutOutlined, UserOutlined } from '@ant-design/icons';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { useSession } from '../api/session';

// The shell is a topbar and a content region, and nothing else: one screen at a time, with the API it
// is talking to and who it thinks it is always visible. At 360px the host hides and the identity
// reduces to an avatar — the two facts that survive are "which server" and "signed in as whom".

export function AppShell({ children, loading }: { children?: ReactNode; loading?: boolean }) {
  const { config, identity, signOut } = useSession();
  return (
    <Flex vertical style={{ minHeight: '100%' }}>
      <header className="noxis-topbar">
        <Link to="/" className="noxis-brand" aria-label="noXis, back to the instances">
          <span className="noxis-brand-dot" aria-hidden="true" />
          noXis
        </Link>
        <span className="noxis-topbar-spacer" />
        {config ? (
          <Tooltip title="The API this console was told to call">
            <span className="noxis-host">{config.api.replace(/^https?:\/\//, '')}</span>
          </Tooltip>
        ) : null}
        {identity ? (
          <Dropdown
            menu={{
              items: [
                {
                  key: 'scopes',
                  label: (
                    <Flex gap={4} wrap style={{ maxWidth: 240 }}>
                      {identity.scopes.map((scope) => (
                        <Tag key={scope}>{scope}</Tag>
                      ))}
                    </Flex>
                  ),
                  disabled: true,
                },
                { type: 'divider' },
                { key: 'signout', icon: <LogoutOutlined />, label: 'Sign out', onClick: signOut },
              ],
            }}
            placement="bottomRight"
          >
            <Flex align="center" gap={8} style={{ cursor: 'pointer' }} role="button" tabIndex={0}>
              <Avatar size={28} style={{ backgroundColor: 'var(--noxis-orange)' }} icon={<UserOutlined />} />
              <span style={{ fontWeight: 600 }}>{identity.name}</span>
            </Flex>
          </Dropdown>
        ) : null}
      </header>
      <main className="noxis-main" style={{ flex: 1 }}>
        {loading ? <Skeleton active paragraph={{ rows: 6 }} /> : children}
      </main>
    </Flex>
  );
}
