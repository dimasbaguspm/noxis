package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/interfaces"
)

const statsInterval = 2 * time.Second

// stats dispatches one engine's stats snapshot. The route is generic; the
// engine's capability decides which shape comes back (SQL, memory cache or
// MongoDB), so the console keeps one stats endpoint per engine.
func (d Deps) stats(ctx context.Context, mod interfaces.Module, r *http.Request) (out any, err error) {
	c, err := d.raw(ctx, mod, r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	switch m := mod.(type) {
	case interfaces.SQLModule:
		return m.Stats(ctx, *c)
	case interfaces.MemcacheModule:
		return m.Stats(ctx, *c)
	case interfaces.MongoModule:
		return m.Stats(ctx, *c, r.URL.Query().Get("database"), r.URL.Query().Get("collection"))
	}
	return nil, unsupported(mod, "stats")
}

// @Summary Instance stats snapshot (connections, size, cache, throughput)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Param database query string false "database to describe (mongodb)"
// @Param collection query string false "collection to describe (mongodb)"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/stats [get]
func GetStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		return d.stats(ctx, mod, r)
	})
}

// @Summary Instance stats stream (SSE, one snapshot every 2s)
// @Param engine path string true "engine family (sql, memcache, mongodb)"
// @Param name path string true "connection name"
// @Success 200 {string} string "text/event-stream"
// @Failure 404 {object} map[string]string
// @Router /v1/{engine}/connections/{name}/stats/stream [get]
func StreamStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	streamSnapshots(w, r, func(ctx context.Context) (any, error) {
		mod, err := d.engine(r)
		if err != nil {
			return nil, err
		}
		return d.stats(ctx, mod, r)
	})
}

// streamSnapshots is the SSE plumbing every family shares: one JSON snapshot
// per tick, failed probes skipped so a blip does not kill the stream.
func streamSnapshots(w http.ResponseWriter, r *http.Request, snapshot func(ctx context.Context) (any, error)) {
	d := From(r)
	ctx, end := d.span(r.Context())
	defer end(nil)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, fmt.Errorf("streaming unsupported"))
		return
	}
	// Resolve the target once: an unknown connection fails before the stream
	// opens, with a normal error body instead of an empty stream.
	if _, err := snapshot(ctx); errors.Is(err, interfaces.ErrNotFound) {
		writeErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()
	for {
		if stats, err := snapshot(ctx); err == nil {
			if body, err := json.Marshal(stats); err == nil {
				if _, err := fmt.Fprintf(w, "event: stats\ndata: %s\n\n", body); err != nil {
					return
				}
				flusher.Flush()
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
