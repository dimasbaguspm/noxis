package master

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
	"noxis/internal/repository"
	rpc "noxis/internal/repository/grpc"
)

type controller struct {
	ctrl *rpc.Server
	repo *repository.SQLite
	log  *slog.Logger
}

type queryReq struct {
	Engine string `json:"engine"`
	SQL    string `json:"sql"`
}

// listStatus merges reported runtime state with the registration registry so
// offline-but-registered nodes stay visible.
func (h *controller) listStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := h.ctrl.Statuses(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	byNode := map[string]entity.NodeStatus{}
	for _, s := range rows {
		byNode[s.Node] = s
	}
	for _, id := range mustListNodes(h, r) {
		if _, ok := byNode[id]; !ok {
			byNode[id] = entity.NodeStatus{Node: id, State: "offline"}
		}
	}
	statuses := make([]entity.NodeStatus, 0, len(byNode))
	for _, s := range byNode {
		statuses = append(statuses, s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": statuses})
}

func mustListNodes(h *controller, r *http.Request) []string {
	ids, err := h.repo.ListIDs(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "list nodes", "error", err)
		return nil
	}
	return ids
}

func (h *controller) getNode(w http.ResponseWriter, r *http.Request) {
	spec, err := h.ctrl.NodeSpec(r.Context(), r.PathValue("node"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, spec)
}

func (h *controller) putNode(w http.ResponseWriter, r *http.Request) {
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
	// publishes the assignment to the node's stream; registered or not, the
	// write is idempotent and the slave converges on its next connect.
	if err := h.ctrl.SetNodeSpec(r.Context(), &entity.Spec{
		Node: r.PathValue("node"), Engines: body.Engines,
	}); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *controller) query(w http.ResponseWriter, r *http.Request) {
	var body queryReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, err)
		return
	}
	out, err := h.ctrl.RouteQuery(r.Context(), r.PathValue("node"), body.Engine, body.SQL)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": out})
}

func (h *controller) register(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Register(r.Context(), r.PathValue("node")); err != nil {
		writeErr(w, err)
		return
	}
	h.log.InfoContext(r.Context(), "node registered", "node", r.PathValue("node"))
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	if strings.Contains(err.Error(), "not connected") {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
