package http

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"noxis/internal/constants"
	"sync"

	"go.yaml.in/yaml/v3"
)

// openapiYAML is what `make spec` produces from the annotations on the handlers, and the only copy of
// the contract. The JSON answer is derived from it at first read so the two can never disagree, and
// a test walks the route table against it so the document cannot drift from the routes either.
//
//go:embed docs/swagger.yaml
var openapiYAML []byte

var (
	openapiJSONOnce sync.Once
	openapiJSONBody []byte
	openapiJSONErr  error
)

// OpenAPIJSON answers the document as JSON.
//
// @Summary The contract, as JSON
// @Description The OpenAPI document this binary serves, derived from the YAML source at first read.
// @Tags system
// @Produce json
// @Success 200 {string} string "the document, not an envelope: this route answers the contract itself"
// @Router /openapi.json [get]
func OpenAPIJSON(w http.ResponseWriter, _ *http.Request) {
	body, err := openAPIJSON()
	if err != nil {
		http.Error(w, "the document does not parse", http.StatusInternalServerError)
		return
	}
	w.Header().Set(constants.HeaderContentType, constants.MediaTypeJSON)
	if _, err := w.Write(body); err != nil {
		return
	}
}

// OpenAPIYAML answers the source, byte for byte: a client that wants to diff the contract gets the
// file the binary was built from.
//
// @Summary The contract, as YAML
// @Description The generated source, exactly as committed.
// @Tags system
// @Produce application/yaml
// @Success 200 {string} string "the document"
// @Router /openapi.yaml [get]
func OpenAPIYAML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(constants.HeaderContentType, constants.MediaTypeYAML)
	if _, err := w.Write(openapiYAML); err != nil {
		return
	}
}

func openAPIJSON() ([]byte, error) {
	openapiJSONOnce.Do(func() {
		var document map[string]any
		if err := yaml.Unmarshal(openapiYAML, &document); err != nil {
			openapiJSONErr = fmt.Errorf("openapi: parse: %w", err)
			return
		}
		openapiJSONBody, openapiJSONErr = json.Marshal(document)
	})
	return openapiJSONBody, openapiJSONErr
}
