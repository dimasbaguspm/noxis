// Package ticker is the standalone scheduler: it POSTs the server's
// /internal/callback/scheduler endpoint on an interval, so due changes run
// without the
// server carrying a scheduler of its own. Run it beside the server, in a
// sidecar, or from cron with a long interval.
package ticker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"noxis/internal/entity"
)

const (
	defaultInterval = 30 * time.Second
	requestTimeout  = 30 * time.Second
)

// Run ticks until ctx is done. The first tick runs immediately, so overdue
// changes are picked up on start (e.g. after a restart).
// schedulerURL is the server the ticker drives. The deployment assumption: a
// single compose stack names the API service "noxis" on port 9000. Tests
// override it; any other deployment edits this constant.
var schedulerURL = "http://noxis:9000/internal/callback/scheduler"

func Run(ctx context.Context, log *slog.Logger, cfg entity.Config) error {
	interval := defaultInterval
	if cfg.TickInterval != "" {
		d, err := time.ParseDuration(cfg.TickInterval)
		if err != nil {
			return fmt.Errorf("%s %q: %w", entity.EnvTickInterval, cfg.TickInterval, err)
		}
		if d <= 0 {
			return fmt.Errorf("%s must be positive", entity.EnvTickInterval)
		}
		interval = d
	}
	log.InfoContext(ctx, "ticker started", "url", schedulerURL, "interval", interval.String())

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		tick(ctx, log, cfg)
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// tick POSTs once and logs the outcome; a failure is logged and retried on the
// next interval, never fatal.
func tick(ctx context.Context, log *slog.Logger, cfg entity.Config) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, schedulerURL, http.NoBody)
	if err != nil {
		log.WarnContext(ctx, "ticker request", "error", err.Error())
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.WarnContext(ctx, "ticker request", "error", err.Error())
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		log.WarnContext(ctx, "ticker tick", "status", resp.StatusCode)
		return
	}
	var res struct {
		Applied []string `json:"applied"`
		Failed  []string `json:"failed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.WarnContext(ctx, "ticker decode", "error", err.Error())
		return
	}
	if len(res.Applied) > 0 || len(res.Failed) > 0 {
		log.InfoContext(ctx, "ticker tick", "applied", len(res.Applied), "failed", len(res.Failed))
	}
}
