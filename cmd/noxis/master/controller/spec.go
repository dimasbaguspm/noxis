package controller

import (
	"encoding/json"
	"errors"
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
)

type putSpecBody struct {
	Engines []entity.Engine `json:"engines"`
}

// @Summary Read a node's desired engines
// @Param node path string true "slave node id"
// @Success 200 {object} entity.Spec
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/spec [get]
func GetNodeSpec(w http.ResponseWriter, r *http.Request) {
	spec, err := From(r).Ctrl.NodeSpec(r.Context(), r.PathValue("node"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

// @Summary Set a node's desired engines (publishes the event to the slave)
// @Param node path string true "slave node id"
// @Param body body putSpecBody true "desired engines"
// @Success 204 "accepted and published"
// @Failure 400 {object} map[string]string
// @Router /v1/nodes/{node}/spec [put]
func PutNodeSpec(w http.ResponseWriter, r *http.Request) {
	var body putSpecBody
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
	// publishes to the node's stream; the slave converges on its next push.
	if err := From(r).Ctrl.SetNodeSpec(r.Context(), &entity.Spec{
		Node: r.PathValue("node"), Engines: body.Engines,
	}); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
