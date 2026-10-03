package handlers

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
)

// runRollback dry-runs a change inside a transaction that always rolls back
// (validate without persisting).
func runRollback(ctx context.Context, dm *doorman.Doorman, node string, eng entity.Engine, req opReq) opRes {
	out, err := dm.RollbackTable(ctx, node, eng, req.SQL)
	if err != nil {
		return opRes{Error: err.Error()}
	}
	return opRes{Output: out}
}
