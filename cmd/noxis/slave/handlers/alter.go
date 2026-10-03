package handlers

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
)

// runAlter applies DDL to the instance (reads/DML rejected by the module's
// kind guard).
func runAlter(ctx context.Context, dm *doorman.Doorman, node string, eng entity.Engine, req opReq) opRes {
	out, err := dm.AlterTable(ctx, node, eng, req.SQL)
	if err != nil {
		return opRes{Error: err.Error()}
	}
	return opRes{Output: out}
}
