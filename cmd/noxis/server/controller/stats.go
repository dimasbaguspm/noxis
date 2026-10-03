package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const statsInterval = 2 * time.Second

// @Summary Instance stats snapshot (connections, size, cache, throughput)
// @Param name path string true "connection name"
// @Success 200 {object} sql.Stats
// @Failure 400 {object} map[string]string
// @Router /v1/sql/connections/{name}/stats [get]
func GetStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.stats(ctx, r.PathValue("name"))
	})
}

// stats collects one snapshot for a named connection.
func (d Deps) stats(ctx context.Context, name string) (out any, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, name)
	if err != nil {
		return nil, err
	}
	return d.SQL.Stats(ctx, *c)
}

// @Summary Instance stats stream (SSE, one snapshot every 2s)
// @Param name path string true "connection name"
// @Success 200 {string} string "text/event-stream"
// @Router /v1/sql/connections/{name}/stats/stream [get]
func StreamStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context())
	defer end(nil)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, fmt.Errorf("streaming unsupported"))
		return
	}
	c, err := d.raw(ctx, r.PathValue("name"))
	if err != nil {
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
		stats, err := d.SQL.Stats(ctx, *c)
		if err == nil {
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
