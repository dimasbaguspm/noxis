# syntax=docker/dockerfile:1
# The console build image: one pinned runtime for the web checks and the bundle.
#   docker build -f apps/web/Dockerfile apps/web                    # runs lint, format, typecheck, build
#   docker build --target dist -f apps/web/Dockerfile apps/web      # exports just the bundle
# The root Dockerfile keeps its own web stage for the full image; this one is
# the contract CI verifies, so a green build here means the same Node and pnpm.
FROM node:22-alpine AS base
ENV PNPM_HOME=/pnpm \
    COREPACK_ENABLE_DOWNLOAD_PROMPT=0 \
    CI=1
ENV PATH=$PNPM_HOME:$PATH
RUN corepack enable && corepack prepare pnpm@10.34.5 --activate
WORKDIR /web

# Dependencies resolve from the lockfile only. The store is pinned to the
# project (alpine cannot hardlink from a foreign filesystem, so pnpm would fall
# back to it anyway) and mounted as a cache so reruns skip the download.
FROM base AS deps
COPY package.json pnpm-lock.yaml .npmrc ./
RUN --mount=type=cache,id=pnpm-store,target=/web/.pnpm-store \
    pnpm install --frozen-lockfile --store-dir .pnpm-store

# Every check runs here, so a formatting, type, or lint failure fails the image.
FROM deps AS build
COPY . .
ENV WEB_OUT_DIR=dist
RUN pnpm lint && pnpm format:check && pnpm build

# The bundle alone, for whoever copies it into the Go embed dir.
FROM scratch AS dist
COPY --from=build /web/dist/ /
