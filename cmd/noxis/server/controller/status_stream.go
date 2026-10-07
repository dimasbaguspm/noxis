package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const statusInterval = 5 * time.Second

// ConnectionState is one instance's reachability, as the dashboard shows it.
type ConnectionState struct {
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latencyMs"`
	Error     string `json:"error,omitempty"`
}

// @Summary Reachability stream for every connection (SSE, one tick per 5s)
// @Success 200 {string} string "text/event-stream"
// @Router /v1/connections/status/stream [get]
func StreamConnectionStatus(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	ctx, end := d.span(r.Context())
	defer end(nil)

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, fmt.Errorf("streaming unsupported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ticker := time.NewTicker(statusInterval)
	defer ticker.Stop()
	for {
		if states, err := d.connectionStates(ctx); err == nil {
			if body, err := json.Marshal(states); err == nil {
				if _, err := fmt.Fprintf(w, "event: status\ndata: %s\n\n", body); err != nil {
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

// connectionStates probes every registered connection (the dashboard only
// renders the ones it shows, but the list is small enough to answer whole).
func (d Deps) connectionStates(ctx context.Context) ([]ConnectionState, error) {
	list, err := d.Connections.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ConnectionState, 0, len(list))
	for _, c := range list {
		state := ConnectionState{Name: c.Name, Driver: c.Driver}
		started := time.Now()
		if mod, ok := d.moduleByDriver(c.Driver); !ok {
			state.Error = "no engine serves driver " + c.Driver
		} else if err := mod.Ping(ctx, c); err != nil {
			state.Error = err.Error()
		} else {
			state.OK = true
		}
		state.LatencyMs = time.Since(started).Milliseconds()
		out = append(out, state)
	}
	return out, nil
}
