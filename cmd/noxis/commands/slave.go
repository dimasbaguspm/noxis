package commands

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"noxis/internal/entity"
	"noxis/internal/features/doorman"
	rpc "noxis/internal/repository/grpc"
)

func newSlaveCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "slave",
		Short: "Run as slave (local reconciler for this node)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.Cfg.Require(entity.EnvMaster, entity.EnvNode); err != nil {
				return err
			}
			return slaveLoop(cmd.Context(), app, app.Cfg.Node)
		},
	}
}

// slaveLoop keeps one Watch stream open, reconciling on every push. On
// disconnect it backs off and reconnects; the master replays the full spec on
// (re)open, so no change is ever missed.
func slaveLoop(ctx context.Context, app *entity.App, node string) error {
	counter := newReconcileCounter()
	for attempt := 0; ; attempt++ {
		if err := watchOnce(ctx, app, node, counter); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			backoff := time.Duration(1<<min(attempt, 5)) * time.Second
			app.Log.WarnContext(ctx, "watch failed",
				"error", err, "retry_in", backoff.String())
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func watchOnce(ctx context.Context, app *entity.App, node string, counter metric.Int64Counter) error {
	conn, err := dial(ctx, app)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := rpc.NewControlClient(conn)
	stream, err := client.Watch(ctx, &rpc.WatchRequest{Node: node})
	if err != nil {
		return err
	}
	app.Log.InfoContext(ctx, "watch connected",
		slog.String("node", node), slog.String("master", app.Cfg.Master))
	reconciler := doorman.NewReconciler(node)
	for {
		spec, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			_, _ = client.ReportStatus(ctx, &rpc.Status{
				Node: node, State: "error", Detail: "disconnected: " + err.Error(),
			})
			return err
		}
		detail, err := reconciler.Reconcile(ctx, rpc.SpecFromProto(spec).Engines)
		state := "ok"
		if err != nil {
			state = "error"
			app.Log.ErrorContext(ctx, "reconcile failed",
				slog.String("node", node), slog.String("error", err.Error()))
			_, _ = client.ReportStatus(ctx, &rpc.Status{Node: node, State: state, Detail: err.Error()})
			continue
		}
		if detail == "" {
			detail = "in sync"
		}
		_, _ = client.ReportStatus(ctx, &rpc.Status{Node: node, State: state, Detail: detail})
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("node", node)))
		app.Log.InfoContext(ctx, "reconciled",
			slog.String("node", node), slog.String("change", detail))
	}
}

func newReconcileCounter() metric.Int64Counter {
	c, _ := otel.Meter("noxis").Int64Counter("noxis.reconcile.total",
		metric.WithDescription("reconciliations performed per node"))
	return c
}
