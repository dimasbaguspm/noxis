package auth

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// Config serves the public auth configuration used by the console bootstrap.
type Config struct {
	usecases.RegistryDeps
	Auth entity.AuthConfig
}

// @Summary Public auth configuration (console bootstrap)
// @Success 200 {object} entity.AuthConfig
// @Router /external/v1/auth/config [get]
func (uc Config) Serve(w http.ResponseWriter, r *http.Request) {
	_, end := logging.StartSpan(r.Context())
	defer end(nil)
	uc.Respond(w, uc.Auth)
}
