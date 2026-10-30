package cache

import (
	"errors"
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Value inspects one key's value.
type Value struct{ usecases.RegistryDeps }

// @Summary Inspect one key's value (rendered per its type)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param name path string true "connection name"
// @Param key query string true "key name"
// @Success 200 {object} entity.Result
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/connections/{name}/value [get]
func (uc Value) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	key := r.URL.Query().Get("key")
	if key == "" {
		uc.Fail(ctx, w, errors.New("key is required"))
		return
	}
	var mc interfaces.KeyBrowser
	if err := uc.RouteModule(ctx, r, &mc); err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	c, err := uc.RawConn(ctx, mc, r.PathValue("name"), entity.LevelRead)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	out, err := mc.Value(ctx, *c, key)
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	uc.Respond(w, out)
}
