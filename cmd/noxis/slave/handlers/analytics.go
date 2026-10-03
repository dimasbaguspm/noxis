package handlers

import (
	"context"

	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
)

// runAnalytics refreshes table statistics (ANALYZE / ANALYZE TABLE).
func runAnalytics(ctx context.Context, dm *doorman.Doorman, node string, eng entity.Engine, req opReq) opRes {
	out, err := dm.AnalyzeTable(ctx, node, eng, req.Table)
	if err != nil {
		return opRes{Error: err.Error()}
	}
	return opRes{Output: out}
}
