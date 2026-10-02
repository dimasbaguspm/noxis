package entity

import "log/slog"

// App is the bootstrap container wired once in main and handed to every
// command: config, logger, and nothing else — commands derive what they need
// from these two.
type App struct {
	Cfg Config
	Log *slog.Logger
}
