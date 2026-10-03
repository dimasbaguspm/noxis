package handlers

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
)

// runQuery runs reads and DML against the instance (DDL rejected by the
// module's kind guard).
func runQuery(ctx context.Context, dm *doorman.Doorman, node string, eng entity.Engine, req opReq) opRes {
	out, err := dm.Query(ctx, node, eng, req.SQL)
	if err != nil {
		return opRes{Error: err.Error()}
	}
	return opRes{Output: out}
}
