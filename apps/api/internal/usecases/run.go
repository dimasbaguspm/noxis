package usecases

import (
	"context"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
)

// RunOp runs the preamble every engine route shares: the span, the typed module for the {driver} in the path, the
// connection at the level the op needs, and the answer. The call supplies what the op does.
//
// useCase names the route in the error metric, which would otherwise read "RunOp" for every failure.
func RunOp[Mod interfaces.Module](d RegistryDeps, w http.ResponseWriter, r *http.Request, useCase, level string, call func(ctx context.Context, module Mod, conn entity.Connection) (any, error)) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	var module Mod
	if err := d.RouteModule(ctx, r, &module); err != nil {
		d.fail(ctx, w, err, useCase)
		return
	}
	conn, err := d.RawConn(ctx, module, r.PathValue("name"), level)
	if err != nil {
		d.fail(ctx, w, err, useCase)
		return
	}
	out, err := call(ctx, module, *conn)
	if err != nil {
		d.fail(ctx, w, err, useCase)
		return
	}
	d.Respond(w, out)
}
