package usecases

import (
	"context"
	"fmt"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/interfaces"
)

// Op is the "Op" use case.
type Op struct {
	RegistryDeps
	Connections interfaces.ConnectionRepository
	Permissions interfaces.PermissionRepository
}

// Serve runs it.
func (uc Op) Run(ctx context.Context, sq interfaces.SQLModule, name, operation, sqlText, table string) (out interfaces.Result, err error) {
	ctx, end := logging.StartSpan(ctx)
	defer func() { end(err) }()
	c, err := rawConn(ctx, uc.Connections, uc.Permissions, uc.Modules, sq, name, OpLevel(operation))
	if err != nil {
		return interfaces.Result{}, err
	}
	switch operation {
	case constants.OpQuery:
		return sq.Query(ctx, *c, sqlText)
	case constants.OpAlter:
		return sq.Alter(ctx, *c, sqlText)
	case constants.OpRollback:
		return sq.DryRun(ctx, *c, sqlText)
	case constants.OpAnalyze:
		return sq.Analyze(ctx, *c, table)
	}
	return interfaces.Result{}, fmt.Errorf("unknown op %q", operation)
}
