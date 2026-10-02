package modules

import (
	"encoding/json"
	"errors"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
)

// registerSpec: GET/PUT /v1/nodes/{node}/spec — read and publish a node's
// desired engines.
func registerSpec(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET /v1/nodes/{node}/spec", func(w http.ResponseWriter, r *http.Request) {
		spec, err := d.Ctrl.NodeSpec(r.Context(), r.PathValue("node"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, spec)
	})

	mux.HandleFunc("PUT /v1/nodes/{node}/spec", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Engines []entity.Engine `json:"engines"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, err)
			return
		}
		for _, e := range body.Engines {
			if _, ok := runtimes.Lookup(e.Type); !ok {
				writeErr(w, errors.New("unknown engine type "+e.Type))
				return
			}
		}
		// publishes to the node's stream; the slave converges on its next
		// push, registered or not.
		if err := d.Ctrl.SetNodeSpec(r.Context(), &entity.Spec{
			Node: r.PathValue("node"), Engines: body.Engines,
		}); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
