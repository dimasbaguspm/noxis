// Package handlers answers the node's op requests — the slave's counterpart
// to the master's controller. The daemon feeds consumed records in; handlers
// resolve the engine from the desired spec, run the doorman op, and reply.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
	"noxis/internal/repositories/kafka"
)

// Deps is everything an op request needs from the daemon.
type Deps struct {
	Log      *slog.Logger
	Producer *kafka.Broker
	Topics   kafka.Topics
	Dm       *doorman.Doorman
}

type opReq struct {
	Op     string `json:"op"` // query | alter | rollback | analytics (doorman.Op*)
	Engine string `json:"engine"`
	SQL    string `json:"sql,omitempty"`
	Table  string `json:"table,omitempty"`
}

type opRes struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// Answer runs the requested op and publishes the reply keyed by the request
// id (the master correlates).
func (d Deps) Answer(ctx context.Context, node string, spec entity.Spec, rec kafka.Record) {
	var req opReq
	if err := json.Unmarshal(rec.Value, &req); err != nil {
		return
	}
	d.Log.InfoContext(ctx, "op received", "op", req.Op, "engine", req.Engine, "key", rec.Key)
	res := opRes{}
	eng, ok := engineOf(spec, req.Engine)
	if !ok {
		res.Error = "no engine named " + req.Engine + " on this node"
	} else {
		res = runOp(ctx, d.Dm, node, req, eng)
	}
	body, err := json.Marshal(res)
	if err != nil {
		return
	}
	if err := d.Producer.PublishSync(ctx, d.Topics.Res(node), rec.Key, body); err != nil {
		d.Log.WarnContext(ctx, "publish op answer", "error", err)
	}
	d.Log.InfoContext(ctx, "op answered", "op", req.Op, "error", res.Error != "")
}

func runOp(ctx context.Context, dm *doorman.Doorman, node string, req opReq, eng entity.Engine) opRes {
	var (
		out string
		err error
	)
	switch req.Op {
	case doorman.OpQuery:
		out, err = dm.Query(ctx, node, eng, req.SQL)
	case doorman.OpAlter:
		out, err = dm.AlterTable(ctx, node, eng, req.SQL)
	case doorman.OpRollback:
		out, err = dm.RollbackTable(ctx, node, eng, req.SQL)
	case doorman.OpAnalytics:
		out, err = dm.AnalyzeTable(ctx, node, eng, req.Table)
	default:
		err = fmt.Errorf("unknown op %q", req.Op)
	}
	if err != nil {
		return opRes{Error: err.Error()}
	}
	return opRes{Output: out}
}

func engineOf(spec entity.Spec, name string) (entity.Engine, bool) {
	for _, e := range spec.Engines {
		if e.Name == name {
			return e, true
		}
	}
	return entity.Engine{}, false
}
