package change

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Approve approves a validated change (self-approval allowed).
type Approve struct{ usecases.RegistryDeps }

func (uc Approve) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	decide(ctx, uc.RegistryDeps, w, r, func(change *entity.Change) error {
		flow, err := flowFor(ctx, change)
		if err != nil {
			return err
		}
		return flow.Approve(ctx, change, entity.Actor(ctx))
	})
}
