package interfaces

import (
	"context"

	"noxis/internal/entity"
)

// AccessRequestRepository persists access requests.
type AccessRequestRepository interface {
	CreateAccessRequest(ctx context.Context, r entity.AccessRequest) error
	PendingRequestExists(ctx context.Context, subject, connection string) (bool, error)
	GetAccessRequest(ctx context.Context, id string) (*entity.AccessRequest, error)
	ListAccessRequests(ctx context.Context, subject string) ([]entity.AccessRequest, error)
	ListPendingRequests(ctx context.Context, connections []string) ([]entity.AccessRequest, error)
	SetAccessRequestDecision(ctx context.Context, id, status, decidedBy string, decidedAt int64, note string) error
}
