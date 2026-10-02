package controller

import (
	"encoding/json"
	"net/http"

	"noxis/internal/entity"
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
	spec, err := From(r).Ctrl.Spec(r.Context(), r.PathValue("node"))
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
	// publishes to the node's topic; the slave acks on its next heartbeat.
	if err := From(r).Ctrl.SetSpec(r.Context(), &entity.Spec{
		Node: r.PathValue("node"), Engines: body.Engines,
	}); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
