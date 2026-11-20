# The console

The console is a static build that ships **inside** the binary. `noxis web serve` serves whatever
was compiled into `internal/web/dist`, on port 3000 unless `--port` says otherwise.

- `internal/web/dist/index.html` and `internal/web/dist/NOT_BUILT` are a committed placeholder. The
  binary serves that page, and says in its log that it is a placeholder, instead of answering 404 to
  the first thing an operator opens.
- A real build replaces `internal/web/dist` and **deletes `NOT_BUILT`**. That file is the only signal
  the binary uses to tell a built console from the placeholder.

## Building

```bash
npm ci && npm run build     # writes dist/
rm -rf ../internal/web/dist && cp -r dist ../internal/web/dist
rm ../internal/web/dist/NOT_BUILT
make -C .. build
```

The console talks to the API over HTTP and knows its address only from `noxis web serve --api`. It
holds no token of its own: signing in is the CLI's job, and the console is served by a process the
operator starts next to the API.
