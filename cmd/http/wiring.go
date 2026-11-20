package main

import (
	"log/slog"
	"net/http"

	"noxis/cmd/http/handlers"
	"noxis/cmd/http/middlewares"
	"noxis/internal/common/config"
)

// deps is what the transport is built from. Adapters and use cases join it as they gain
// a route; this struct is the only place they are named.
type deps struct {
	cfg      config.Config
	log      *slog.Logger
	auth     *middlewares.Authenticator
	handlers *handlers.Handlers
}

// build is the composition root: config → logger → adapters → use cases → transport.
func build(cfg config.Config, log *slog.Logger) (http.Handler, error) {
	auth, err := middlewares.NewAuthenticator(cfg.Tokens, log)
	if err != nil {
		return nil, err
	}
	if len(cfg.SecretKey) == 0 {
		log.Warn("SECRET_KEY is unset: a target that has a password cannot be created until it is")
	}
	return newMux(deps{cfg: cfg, log: log, auth: auth, handlers: handlers.New(log)}), nil
}
