// Package rest is the master's public API: frontends and the CLI hit this,
// the master publishes events over the internal gRPC river to slave workers.
package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"noxis/internal/entity"
	"noxis/internal/features/doorman/runtimes"
	rpc "noxis/internal/repository/grpc"
)

type queryReq struct {
	Engine string `json:"engine"`
	SQL    string `json:"sql"`
}

type putSpecReq struct {
	Engines []entity.Engine `json:"engines"`
}

type NodeStatusJSON struct {
	Node   string `json:"node"`
	State  string `json:"state"`
	Detail string `json:"detail"`
	Ts     int64  `json:"ts"`
}

// New returns the master's HTTP handler.
func New(ctrl *rpc.Server) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/status", func(w http.ResponseWriter, r *http.Request) {
		rows, err := ctrl.Statuses(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		statuses := make([]NodeStatusJSON, 0, len(rows))
		for _, s := range rows {
			statuses = append(statuses, NodeStatusJSON{Node: s.Node, State: s.State, Detail: s.Detail, Ts: s.TS})
		}
		writeJSON(w, http.StatusOK, map[string]any{"nodes": statuses})
	})

	mux.HandleFunc("GET /v1/nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		spec, err := ctrl.NodeSpec(r.Context(), r.PathValue("node"))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, spec)
	})

	mux.HandleFunc("PUT /v1/nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		var body putSpecReq
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
		spec := &entity.Spec{Node: r.PathValue("node"), Engines: body.Engines}
		if err := ctrl.SetNodeSpec(r.Context(), spec); err != nil { // publishes to the node's stream
			writeErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /v1/nodes/{node}/query", func(w http.ResponseWriter, r *http.Request) {
		var body queryReq
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeErr(w, err)
			return
		}
		out, err := ctrl.RouteQuery(r.Context(), r.PathValue("node"), body.Engine, body.SQL)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"output": out})
	})

	return allowCORS(mux)
}

func allowCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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
