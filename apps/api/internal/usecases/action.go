package usecases

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"noxis/internal/entity"
)

// ActionHandler dispatches a workflow action to the use case that owns it.
// It reads the action, then replays the body to the chosen handler: a
// transition payload is small and the target still decodes it itself.
func ActionHandler(actions map[string]http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			writeActionError(w, "cannot read the action payload")
			return
		}
		var body entity.TransitionRequest
		if err := json.Unmarshal(raw, &body); err != nil {
			writeActionError(w, "action payload must be JSON")
			return
		}
		next, ok := actions[body.Action]
		if !ok {
			writeActionError(w, "unknown action "+body.Action)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		next.ServeHTTP(w, r)
	}
}

func writeActionError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(entity.ErrorResponse{Error: message})
}
