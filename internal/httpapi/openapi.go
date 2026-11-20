package httpapi

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"noxis/internal/constants"
	"sync"

	"go.yaml.in/yaml/v3"
)

// openapiYAML is the contract, and the only copy of it. The JSON answer is derived at first read so
// the two can never disagree, and a test walks the route table against this file so the document
// cannot drift from the routes either.
//
//go:embed openapi.yaml
var openapiYAML []byte

var (
	openapiJSONOnce sync.Once
	openapiJSONBody []byte
	openapiJSONErr  error
)

// OpenAPIJSON answers the document as JSON.
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
