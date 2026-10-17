package metadata

import (
	"net/http"
	"strings"

	"noxis/internal/usecases"
)

// Tracing exposes the OTLP endpoint the console may export to. Only an https
// endpoint is served: a browser cannot reach a plaintext collector from an
// https page, and an internal host must not leak to the client. Anything else
// means the console leaves browser tracing off.
type Tracing struct {
	usecases.RegistryDeps
	Endpoint string
}

// @Summary OTLP endpoint the console may export traces to
// @Success 200 {object} object{endpoint=string}
// @Router /external/v1/config/tracing [get]
func (uc Tracing) Serve(w http.ResponseWriter, _ *http.Request) {
	endpoint := ""
	if strings.HasPrefix(uc.Endpoint, "https://") {
		endpoint = uc.Endpoint
	}
	uc.Respond(w, map[string]any{"endpoint": endpoint})
}
