import { ArrowUpOutlined } from '@ant-design/icons';
import { Tooltip } from 'antd';
import { useVersion } from '../api/queries';
import { AppButton } from './app-button';
import { Text } from './text';

// The build is a fact worth showing: `nightly+db92d5a` is what a bug report should quote, and it is the same
// string the CLI prints. The newest published build is the server's answer — it caches the check for an hour,
// so opening ten tabs costs one request — and every link points at the releases page, never at a raw asset.
//
// Both live in the top bar and nowhere else: a deployment that is one build behind is not an incident, and a
// banner across every page is one. The chip says what this build is; the button, only when there is
// something to move to, says where.

/** The build this deployment answers as, linking at the releases page. */
export function BuildBadge() {
  const version = useVersion();
  if (!version.data) return null;

  // The server already answers `nightly+db92d5a` or `dev+db92d5a`: the release line and the commit are one
  // string, and it is the same one `noxis version` prints.
  const build = version.data.version;

  return (
    <Tooltip title={`This deployment answers as ${build}`}>
      <a
        href={version.data.release_url}
        target="_blank"
        rel="noreferrer"
        className="hidden rounded-md border border-line px-2 py-0.5 transition-colors hover:border-brand-ink sm:inline"
      >
        <Text mono size="xs" tone="muted">
          {build}
        </Text>
      </a>
    </Tooltip>
  );
}

/** A newer build is published: one button, pointing at the releases page, and nothing else changes. */
export function UpgradeButton() {
  const version = useVersion();
  const latest = version.data?.latest;
  if (!version.data?.newer || !latest) return null;

  const published = `${latest.version}+${latest.commit}`;

  return (
    <Tooltip
      title={`${published} is published; this deployment answers as ${version.data.version}`}
    >
      {/* An anchor through the button, not a link styled by hand: Ant Design's reset owns `a { color }` and
          `a { background }`, and a layered Tailwind utility loses to an unlayered rule every time. */}
      <AppButton
        size="small"
        intent="primary"
        href={version.data.release_url}
        target="_blank"
        rel="noreferrer"
        icon={<ArrowUpOutlined />}
        aria-label={`Upgrade to latest: ${published} is published`}
      >
        <span className="hidden sm:inline">Upgrade to latest</span>
        <span className="sm:hidden">Upgrade</span>
      </AppButton>
    </Tooltip>
  );
}
