package usecases

import (
	"slices"

	"noxis/internal/constants"
	"noxis/internal/interfaces"
)

// Serves reports whether a module declares a capability. A module answers every verb, so "can this driver do it" is read
// from its declaration instead of from a type assertion that is true for every module.
func Serves(module interfaces.Module, capability string) bool {
	return slices.Contains(module.Capabilities(), capability)
}

// ServesKind reports whether a module declares the capability a change kind needs: the kind picks the family, and the
// family has to be one the driver claims.
func ServesKind(module interfaces.Module, kind string) bool {
	capability := capabilityOfKind(kind)
	return capability != "" && Serves(module, capability)
}

// capabilityOfKind maps a change kind to the capability its engine must declare, or "" for a kind nobody classifies.
func capabilityOfKind(kind string) string {
	switch kind {
	case constants.KindDML, constants.KindDDL:
		return constants.CapabilitySQL
	case constants.KindMongo, constants.KindDocument:
		return constants.CapabilityDocuments
	case constants.KindSearch:
		return constants.CapabilitySearch
	}
	return ""
}
