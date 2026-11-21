import { Avatar, Dropdown, Tag, Tooltip } from 'antd';
import { DownOutlined, LogoutOutlined } from '@ant-design/icons';
import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useSession } from '../api/session-context';
import { Logo } from '../components/logo';
import { Text } from '../components/text';
import { UpdateNotice, VersionChip } from '../components/update-notice';

// A topbar and a content region, and nothing else. At 360px the host hides and the identity reduces to
// an avatar; the two facts that survive are which server, and who you are.

export function AppShell({ children, anonymous }: { children: ReactNode; anonymous?: boolean }) {
  const { config, identity, signOut } = useSession();
  const { pathname } = useLocation();
  const home = pathname === '/';

  return (
    <div className="flex min-h-full flex-col">
      <a
        href="#content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:rounded focus:bg-white focus:px-3 focus:py-2"
      >
        Skip to content
      </a>

      <header className="sticky top-0 z-10 flex h-14 items-center gap-3 border-b border-line bg-white/95 px-4 backdrop-blur md:px-6">
        <Link
          to="/"
          className="flex items-center gap-2 text-base font-bold tracking-tight"
          aria-label="noXis, back to the instances"
          aria-current={home ? 'page' : undefined}
        >
          <Logo size={26} />
        </Link>

        <span className="flex-1" />

        {config ? (
          <Tooltip title="The API this console was told to call">
            <Text
              mono
              size="xs"
              tone="muted"
              className="hidden max-w-[40vw] truncate rounded-md border border-line px-2 py-0.5 sm:inline"
            >
              {config.api.replace(/^https?:\/\//, '')}
            </Text>
          </Tooltip>
        ) : null}

        <VersionChip />

        {identity ? (
          <Dropdown
            trigger={['click']}
            placement="bottomRight"
            menu={{
              items: [
                {
                  key: 'who',
                  disabled: true,
                  label: (
                    <div className="flex max-w-64 flex-col gap-1 py-0.5">
                      <Text className="font-semibold">{identity.name}</Text>
                      <Text mono size="xs" tone="muted" className="truncate">
                        {config?.api.replace(/^https?:\/\//, '') ?? 'no API'}
                      </Text>
                    </div>
                  ),
                },
                {
                  key: 'scopes',
                  disabled: true,
                  label: (
                    <div className="flex max-w-64 flex-wrap gap-1">
                      {identity.scopes.map((scope) => (
                        <Tag key={scope} className="m-0">
                          {scope}
                        </Tag>
                      ))}
                    </div>
                  ),
                },
                { type: 'divider' },
                { key: 'out', icon: <LogoutOutlined />, label: 'Sign out', onClick: signOut },
              ],
            }}
          >
            <button
              type="button"
              className="flex items-center gap-2 rounded-full border border-line py-0.5 pr-2.5 pl-0.5 transition-colors hover:border-brand hover:bg-brand-soft"
              aria-label={`signed in as ${identity.name}, open the session menu`}
            >
              <Avatar size={26} className="!bg-brand-ink !text-[11px] !font-semibold">
                {initials(identity.name)}
              </Avatar>
              <Text className="hidden font-semibold sm:inline">{identity.name}</Text>
              <DownOutlined className="hidden text-[10px] text-muted sm:inline" />
            </button>
          </Dropdown>
        ) : anonymous ? (
          <Tag>signed out</Tag>
        ) : null}
      </header>

      <main id="content" className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 md:px-8 md:py-8">
        <UpdateNotice />
        {children}
      </main>
    </div>
  );
}

/** Two letters for the avatar: a token name is a word, and a word in a 26px circle is a smudge. */
function initials(name: string): string {
  const letters = name.replace(/[^a-zA-Z0-9]/g, '');
  return letters.slice(0, 2).toUpperCase() || '?';
}
