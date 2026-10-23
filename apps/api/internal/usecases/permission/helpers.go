package permission

import (
	"context"

	"noxis/internal/constants"
	"noxis/internal/entity"
)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func adminLevel(ctx context.Context, need string) (string, error) {
	principal, ok := entity.Principal(ctx)
	if !ok {
		return "", constants.ErrUnauthorized
	}
	if principal.IsAdmin() {
		return "", nil
	}
	return need, nil
}
