package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"noxis/internal/entity"
	"noxis/internal/features/doorman"
	"noxis/internal/features/doorman/runtimes"
	rpc "noxis/internal/repository/grpc"
)

// status words and backoff policy.
const (
	stateOK         = "ok"
	stateError      = "error"
	detailSync      = "in sync"
	detailDead      = "disconnected: "
	maxBackoffShift = 5 // backoff = 2^n s, 32s ceiling
	reconcileMetric = "noxis.reconcile.total"
)

func newSlaveCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "slave",
		Short: "Run as slave (local reconciler for this node)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Cfg.Require(entity.EnvGRPC, entity.EnvNode); err != nil {
				return err
			}
			return slaveLoop(cmd.Context(), app, app.Cfg.Node)
		},
	}
}

// slaveLoop keeps the bidirectional stream alive; the master replays the full
// spec on every (re)connect, so no change is ever missed.
func slaveLoop(ctx context.Context, app *entity.App, node string) error {
	counter := newReconcileCounter()
	for attempt := 0; ; attempt++ {
		if err := watchOnce(ctx, app, node, counter); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			backoff := time.Duration(1<<min(attempt, maxBackoffShift)) * time.Second
			app.Log.WarnContext(ctx, "watch failed",
				slog.String("error", err.Error()), slog.Duration("retry_in", backoff))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func watchOnce(ctx context.Context, app *entity.App, node string, counter metric.Int64Counter) error {
	conn, err := dial(ctx, app.Cfg.GRPC)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	client := rpc.NewControlClient(conn)
	stream, err := client.Watch(ctx)
	if err != nil {
		return fmt.Errorf("watch rpc: %w", err)
	}
	if err := stream.Send(&rpc.StreamMsg{Node: node}); err != nil { // hello: bind the stream
		return fmt.Errorf("hello: %w", err)
	}
	app.Log.InfoContext(ctx, "watch connected",
		slog.String("node", node), slog.String("master", app.Cfg.GRPC))

	reconciler := doorman.NewReconciler(node)
	var engines []entity.Engine // last desired slice, for query resolution

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			_, _ = client.ReportStatus(ctx, &rpc.Status{
				Node: node, State: stateError, Detail: detailDead + err.Error(),
			})
			return err
		}

		switch p := msg.Payload.(type) {
		case *rpc.StreamMsg_Spec:
			engines = rpc.SpecFromProto(p.Spec).Engines
			detail, err := reconciler.Reconcile(ctx, engines)
			state := stateOK
			if err != nil {
				state = stateError
				app.Log.ErrorContext(ctx, "reconcile failed",
					slog.String("node", node), slog.String("error", err.Error()))
				_, _ = client.ReportStatus(ctx, &rpc.Status{Node: node, State: state, Detail: err.Error()})
				continue
			}
			if detail == "" {
				detail = detailSync
			}
			_, _ = client.ReportStatus(ctx, &rpc.Status{Node: node, State: state, Detail: detail})
			counter.Add(ctx, 1, metric.WithAttributes(attribute.String("node", node)))
			app.Log.InfoContext(ctx, "reconciled",
				slog.String("node", node), slog.String("change", detail))

		case *rpc.StreamMsg_Query:
			res := executeQuery(ctx, node, engines, p.Query)
			if err := stream.Send(&rpc.StreamMsg{
				Node: node, Payload: &rpc.StreamMsg_Result{Result: res},
			}); err != nil {
				return err
			}
		}
	}
}

// executeQuery runs a read query in the engine's container. The engine must
// be in the node's last desired slice — queries only touch engines this node
// owns, via the same credentials the container was created with.
func executeQuery(ctx context.Context, node string, engines []entity.Engine, q *rpc.QueryRequest) *rpc.QueryResult {
	for _, e := range engines {
		if e.Name != q.Engine {
			continue
		}
		out, err := runtimes.ExecSQL(ctx, runtimes.ContainerName(node, e.Name), e, q.Sql)
		if err != nil {
			return &rpc.QueryResult{Id: q.Id, Error: err.Error()}
		}
		return &rpc.QueryResult{Id: q.Id, Output: out}
	}
	return &rpc.QueryResult{Id: q.Id, Error: errors.New("no engine named " + q.Engine + " on this node").Error()}
}

func newReconcileCounter() metric.Int64Counter {
	c, _ := otel.Meter("noxis").Int64Counter(reconcileMetric,
		metric.WithDescription("reconciliations performed per node"))
	return c
}
