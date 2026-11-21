# The console

React + TypeScript + Ant Design, built by Vite straight into `internal/web/dist`, which the binary
embeds and `http serve` mounts at `/`. One command builds the console the image carries, and one
process serves it:

```bash
pnpm -C web install     # once; the lockfile is committed and never rewritten by CI
make web                # or: pnpm -C web run build
make build              # the binary now carries it
./bin/noxis http serve  # open http://127.0.0.1:9000 — the page and the API share an origin
```

`pnpm dev` is the exception: Vite serves the page on `:5173` and the API is somewhere else, so that
deployment is the one that names `CONSOLE_ORIGINS`. A same-origin console needs nothing.

**pnpm, and pinned versions.** `web/package.json` names an exact version of every dependency and of
pnpm itself (`packageManager`), and `web/.npmrc` sets `save-exact`, so `pnpm add x` writes `1.2.3` and
an upgrade is a commit somebody reviewed. `pnpm-lock.yaml` is committed; the image installs with
`--frozen-lockfile`.

`dist/NOT_BUILT` is the only signal that separates a committed placeholder from a real build. A build
empties the directory, so the marker disappears on its own; the placeholder exists so a binary built
without pnpm still answers something a human can read. **A local build is never committed**: restore
`internal/web/dist/index.html` and `NOT_BUILT` before you commit.

## Shape

| Path                          | What it is                                                             |
| ----------------------------- | ---------------------------------------------------------------------- |
| `src/api/http.ts`             | the axios instance, the envelope, the error a screen can show          |
| `src/api/queries.ts`          | one react-query hook per read and per action                           |
| `src/api/session-context.tsx` | the API it found, the identity `/v1/whoami` confirmed, sign in and out |
| `src/api/permissions.tsx`     | scopes as values: `usePermissions`, `<Can do="approve">`               |
| `src/app/`                    | the shell (`app-shell.tsx`) and the split sign-in screen               |
| `src/components/`             | Ant Design, wrapped once: `alert`, `data-table`, `form`, `inputs`      |
| `src/editor/sql-editor.tsx`   | Monaco, bundled (never a CDN), completing from the target's own schema |
| `src/pages/targets-page.tsx`  | the list, the search, the create drawer                                |
| `src/pages/target-page.tsx`   | the name, connectivity, and the Query/Tables/Changes/Details routes    |
| `src/pages/proposal-page.tsx` | the statement, the rollback, the rehearsal, review and apply           |

## Theme

Orange `#f97316` on white, one grey scale, 8px radius. Tailwind's `@theme` block in `styles.css` and
Ant Design's tokens in `theme.ts` name the same values, and nothing else in the console writes a
colour. Component-level fixes (the bigger `Alert`, the one table style) live in `styles.css` under a
`noxis-` class, so a page cannot style Ant Design directly.

## What the console does not do yet

- It has no ERD tab: the API exposes columns and indexes, not foreign keys.
- It has no frontend test runner yet (#824).
