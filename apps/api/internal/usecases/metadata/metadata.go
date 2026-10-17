// Package metadata holds the read-only catalogs the console bootstraps from, so
// the frontend follows the backend's values instead of keeping its own copy.
package metadata

import (
	"net/http"

	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// LevelInfo is one access level and its human label.
type LevelInfo struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// levels is the single source for the console's level pickers.
var levels = []LevelInfo{
	{Value: entity.LevelRead, Label: "Read"},
	{Value: entity.LevelWrite, Label: "Editor"},
	{Value: entity.LevelAdmin, Label: "Admin"},
}

// Levels returns the access levels the API understands.
type Levels struct{ usecases.RegistryDeps }

// @Summary Access levels and their labels
// @Success 200 {object} object{levels=[]LevelInfo}
// @Router /external/v1/config/levels [get]
func (uc Levels) Serve(w http.ResponseWriter, _ *http.Request) {
	uc.Respond(w, map[string]any{"levels": levels})
}
