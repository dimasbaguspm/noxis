package uat

import (
	"context"
	"fmt"
	"net/http"
)

// serverInfoBody is the connection info response the instance page shows.
type serverInfoBody struct {
	Engine       string   `json:"engine"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// serverReportsVersion proves the version comes from the server rather than from a field an operator typed: every driver answers with one, plus the capabilities that follow from its profile.
func (w *world) serverReportsVersion(context.Context) error {
	var info serverInfoBody
	if err := w.api(http.MethodGet, w.enginePath("/connections/"+w.conn+"/info"), nil, &info); err != nil {
		return err
	}
	if w.code != http.StatusOK {
		return fmt.Errorf("server info: %d %s", w.code, w.body)
	}
	if info.Engine == "" || info.Version == "" {
		return fmt.Errorf("server info must name the engine and the version it reported: %s", w.body)
	}
	if len(info.Capabilities) == 0 {
		return fmt.Errorf("server info must state what the profile can prove: %s", w.body)
	}
	return nil
}
