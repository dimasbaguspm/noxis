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

// @Summary Instance stats snapshot (connections, size, cache, throughput)
// @Param name path string true "connection name"
// @Success 200 {object} sql.Stats
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/stats [get]
func GetStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.SQL, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.SQL.Stats(ctx, *c)
	})
}

// @Summary Instance stats stream (SSE, one snapshot every 2s)
// @Param name path string true "connection name"
// @Success 200 {string} string "text/event-stream"
// @Router /v1/sql/connections/{name}/stats/stream [get]
func StreamStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	streamSnapshots(w, r, func(ctx context.Context) (any, error) {
		c, err := d.raw(ctx, d.SQL, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.SQL.Stats(ctx, *c)
	})
}

// streamSnapshots is the SSE plumbing both families share: one JSON snapshot
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
