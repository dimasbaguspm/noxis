package modules

import (
	"net/http"

	"noxis/doc"
)

// registerDocs wires the OpenAPI spec route.
func registerDocs(mux *http.ServeMux, _ Deps) {
	mux.HandleFunc("GET /openapi.json", getOpenAPIJSON)
	mux.HandleFunc("GET /openapi.yaml", getOpenAPIYAML)
}

// @Summary This OpenAPI spec (json)
// @Success 200 {string} string
// @Router /openapi.json [get]
func getOpenAPIJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(doc.SwaggerJSON)
}

// @Summary This OpenAPI spec (yaml)
// @Success 200 {string} string
// @Router /openapi.yaml [get]
func getOpenAPIYAML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(doc.SwaggerYAML)
}
