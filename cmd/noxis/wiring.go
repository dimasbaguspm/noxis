package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	apiserver "noxis/cmd/noxis/http"
	"noxis/cmd/noxis/http/handlers"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/internal/adapters/mysql"
	"noxis/internal/adapters/sqlite"
	"noxis/internal/app/proposal"
	"noxis/internal/app/target"
	"noxis/internal/common/config"
	"noxis/internal/common/keyed"
	"noxis/internal/common/release"
	"noxis/internal/common/secret"
	"noxis/internal/constants"
	"noxis/internal/web"
	"time"

	appread "noxis/internal/app/read"
)

// buildServer is the composition root: config → logger → adapters → use cases → transport. It is
// the only place in the tree that names a concrete adapter, and it returns the closer for the state
// file so the caller can shut things down in order.
func buildServer(ctx context.Context, cfg config.Config, statePath string, log *slog.Logger) (http.Handler, io.Closer, error) {
	locks := &keyed.Group{}
	state, err := sqlite.Open(ctx, statePath)
	if err != nil {
		return nil, nil, err
	}
	sealer, err := secret.NewWithPrevious(cfg.SecretKey, cfg.SecretKeyOld...)
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
		Proposals:    state,
		Rehearsals:   state,
		Targets:      state,
		Journal:      state,
		Sealer:       sealer,
		Driver:       driver,
		Timeout:      cfg.QueryTimeout,
		ProbeTimeout: cfg.ProbeTimeout,
		Locks:        locks,
	}
	reads := appread.Deps{
		Targets: state,
		Sealer:  sealer,
		Driver:  driver,
		Timeout: cfg.QueryTimeout,
	}
	handlers_ := handlers.New(handlers.Deps{
		Log:     log,
		Release: release.NewChecker(constants.VersionAsset, time.Hour),
		Targets: handlers.TargetUseCases{
			Create:  target.NewCreate(useCases),
			List:    target.NewList(useCases),
			Get:     target.NewGet(useCases),
			Update:  target.NewUpdate(useCases),
			Archive: target.NewArchive(useCases),
			Probe:   target.NewProbe(useCases),
			Test:    target.NewTest(useCases),
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

	// The console is not a second deployment any more: it is this origin's front page, so the browser
	// calls the API it was served from and no CORS decision is involved. `/v1` is the prefix the route
	// table writes; the page needs it only to know where the calls go.
	console, err := web.New(web.Config{API: "/v1"})
	if err != nil {
		_ = state.Close()
		return nil, nil, err
	}
	// One line says where the page is and which of the two it is serving: a 404 on the first thing an
	// operator opens is a bug report about the wrong thing.
	log.Info("console", "path", "/", "built", console.Built())

	handler := apiserver.NewMux(apiserver.Deps{
		Log:      log,
		Auth:     auth,
		Handlers: handlers_,
		Origins:  cfg.ConsoleOrigins,
		Console:  console.Handler(),
	})
	return handler, state, nil
}
