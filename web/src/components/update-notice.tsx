import { Tooltip } from 'antd';
import { useState } from 'react';
import { useVersion } from '../api/queries';
import { Alert } from './alert';
import { AppButton } from './app-button';
import { RenderWhen } from './render-when';
import { Text } from './text';

// The build is a fact worth showing: `nightly+db92d5a` is what a bug report should quote, and it is the same
// string the CLI prints. The newest published build is the server's answer — it caches the check for an hour,
// so opening ten tabs costs one request — and every notice points at the releases page, never at a raw asset.

export function VersionChip() {
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
        className="hidden rounded-md border border-line px-2 py-0.5 md:inline"
      >
        <Text mono size="xs" tone="muted">
          {build}
        </Text>
      </a>
    </Tooltip>
  );
}

export function UpdateNotice() {
  const version = useVersion();
  const [dismissed, setDismissed] = useState(false);
  const latest = version.data?.latest;

  return (
    <RenderWhen condition={version.data?.newer && !dismissed && latest}>
      <Alert
        tone="info"
        className="mb-4"
        title={`A newer build is published: ${latest?.version}+${latest?.commit}`}
        description={
          <Text size="sm" tone="muted">
            This deployment answers as {version.data?.version}. Updates are downloaded and installed
            by hand, from the releases page.
          </Text>
        }
        action={
          <span className="flex gap-2">
            <AppButton
              size="small"
              intent="primary"
              onClick={() => window.open(version.data?.release_url, '_blank', 'noreferrer')}
            >
              See the release
            </AppButton>
            <AppButton size="small" intent="quiet" onClick={() => setDismissed(true)}>
              Dismiss
            </AppButton>
          </span>
        }
      />
    </RenderWhen>
  );
}
