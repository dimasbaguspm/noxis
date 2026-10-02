package modules

import (
	"net/http"

	"noxis/doc"
)

// registerDocs: GET /openapi.yaml serves the embedded OpenAPI spec (no UI).
func registerDocs(mux *http.ServeMux, _ Deps) {
	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(doc.SwaggerYAML)
	})
}