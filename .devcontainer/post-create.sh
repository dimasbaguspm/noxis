#!/usr/bin/env bash
# Devcontainer setup: match the pinned toolchain and install both apps' deps.
# Go version comes from apps/api/go.mod, Node from apps/web/package.json engines,
# pnpm from its "packageManager" field.
set -euo pipefail
cd "$(dirname "$0")/.."

corepack enable
corepack prepare pnpm@10.34.5 --activate

(cd apps/api && go mod download)
(cd apps/web && pnpm install --frozen-lockfile)

echo "noxis devcontainer ready: 'make dev' (stack) or 'make dev-ui' (+ hot console)"
