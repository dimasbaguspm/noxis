# The console

React + TypeScript + Ant Design, built by Vite straight into `internal/web/dist`, which the binary
embeds and serves with `noxis web serve`. One command builds the console the image carries:

```bash
npm ci                 # once
make -C .. web         # or: npm run build
make -C .. build       # the binary now carries it
noxis web serve --api https://api.example.com
```

`dist/NOT_BUILT` is the only signal that separates a committed placeholder from a real build. A build
empties the directory, so the marker disappears on its own; the placeholder exists so a binary built
without npm still answers something a human can read.

## Shape

| Path | What it is |
| --- | --- |
| `src/api/client.ts` | the envelope, the bearer header, the token cookie, one function per route |
| `src/api/session.tsx` | the API it found, the identity `/v1/whoami` confirmed, sign in and out |
| `src/app/` | the shell (topbar + content) and the sign-in card |
| `src/pages/TargetsPage.tsx` | the list, the search, the create drawer |
| `src/pages/TargetPage.tsx` | Query, DML, DDL, History for one target |
| `src/pages/ProposalPage.tsx` | the statement, the rollback, the rehearsal, review and apply |
| `src/editor/SqlEditor.tsx` | Monaco, bundled (never a CDN), completing from the target's own schema |

## Theme

Orange `#f97316` on white, one grey scale, 8px radius. Every colour comes from `src/theme.ts`; a screen
that needs a colour imports it from there rather than writing a hex literal.

## What the console does not do yet

- It has no ERD tab: the API exposes columns and indexes, not foreign keys.
- It signs in with a pasted token; the CLI's stored session is not handed to the browser (#821).
