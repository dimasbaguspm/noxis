package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"noxis/cmd/http/handlers"
	"noxis/cmd/http/middlewares"
	"noxis/internal/adapters/mysql"
	"noxis/internal/adapters/sqlite"
	"noxis/internal/app/proposal"
	appread "noxis/internal/app/read"
	"noxis/internal/app/target"
	"noxis/internal/common/config"
	"noxis/internal/common/secret"
)

// deps is what the transport is built from. This struct and build() are the only places a
// concrete adapter is named: nothing below the transport knows the state file exists.
type deps struct {
	cfg      config.Config
	log      *slog.Logger
	auth     *middlewares.Authenticator
	handlers *handlers.Handlers
}

// build is the composition root: config → logger → adapters → use cases → transport. It
// returns the closer for the state file so the caller can shut it down in order.
func build(ctx context.Context, cfg config.Config, log *slog.Logger) (http.Handler, io.Closer, error) {
	state, err := sqlite.Open(ctx, cfg.StatePath)
	if err != nil {
		return nil, nil, err
	}
	sealer, err := secret.New(cfg.SecretKey, cfg.SecretKeyPrevious)
	if err != nil {
		_ = state.Close()
		return nil, nil, err
	}
	auth, err := middlewares.NewAuthenticator(cfg.Tokens, log)
	if err != nil {
		_ = state.Close()
		return nil, nil, err
	}
	if !sealer.Enabled() {
		log.Warn("SECRET_KEY is unset: a target that carries a password cannot be registered until it is")
	}

	driver := mysql.New(cfg.ProbeTimeout)
	useCases := target.Deps{
		Targets: state,
		Journal: state,
		Sealer:  sealer,
		Driver:  driver,
	}
	proposals := proposal.Deps{
		Proposals:  state,
		Rehearsals: state,
		Targets:    state,
		Journal:    state,
		Sealer:     sealer,
		Driver:     driver,
		Timeout:    cfg.QueryTimeout,
	}
	reads := appread.Deps{
		Targets: state,
		Sealer:  sealer,
		Driver:  driver,
		Timeout: cfg.QueryTimeout,
	}
	handlers_ := handlers.New(handlers.Deps{
		Log: log,
		Targets: handlers.TargetUseCases{
			Create:  target.NewCreate(useCases),
			List:    target.NewList(useCases),
			Get:     target.NewGet(useCases),
			Update:  target.NewUpdate(useCases),
			Archive: target.NewArchive(useCases),
			Probe:   target.NewProbe(useCases),
		},
		Read: handlers.ReadUseCases{
			Query:     appread.NewQuery(reads),
			Databases: appread.NewDatabases(reads),
			Tables:    appread.NewTables(reads),
			Table:     appread.NewTable(reads),
		},
		Proposals: handlers.ProposalUseCases{
			Create:   proposal.NewCreate(proposals),
			List:     proposal.NewList(proposals),
			Get:      proposal.NewGet(proposals),
			Rehearse: proposal.NewRehearse(proposals),
			Decide:   proposal.NewDecide(proposals),
			Apply:    proposal.NewApply(proposals),
			Events:   proposal.NewEvents(proposals),
		},
	})

	handler := newMux(deps{cfg: cfg, log: log, auth: auth, handlers: handlers_})
	return handler, state, nil
}
