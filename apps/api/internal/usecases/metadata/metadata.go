// Package metadata holds the read-only catalogs the console bootstraps from, so the frontend follows the backend's values instead of keeping its own copy.
package metadata

import (
	"net/http"

	"noxis/internal/common/logging"
	"noxis/internal/entity"
	"noxis/internal/usecases"
)

// levels is the single source for the console's level pickers.
var levels = []entity.LevelInfo{
	{Value: entity.LevelRead, Label: "Read"},
	{Value: entity.LevelWrite, Label: "Editor"},
	{Value: entity.LevelAdmin, Label: "Admin"},
}

// Levels returns the access levels the API understands.
type Levels struct{ usecases.RegistryDeps }

// @Summary Access levels and their labels
// @Success 200 {object} entity.LevelList
// @Router /external/v1/config/levels [get]
func (uc Levels) Serve(w http.ResponseWriter, r *http.Request) {
	_, end := logging.StartSpan(r.Context())
	defer end(nil)
	uc.Respond(w, entity.LevelList{Levels: levels})
}
