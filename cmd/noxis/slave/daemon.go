package slave

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

const (
	stateOK         = "ok"
	stateError      = "error"
	detailSync      = "in sync"
	detailDead      = "disconnected: "
	maxBackoffShift = 5
	reconcileMetric = "noxis.reconcile.total"
)

// NewRunCmd is `noxis run slave <id>`: starts the worker daemon from the
// identity file written by register. The id must match the identity.
func NewRunCmd(app *entity.App) *cobra.Command {
	return &cobra.Command{
		Use:   "slave [id]",
		Short: "Run the slave worker daemon for this machine (needs registered identity)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := Load()
			if err != nil {
				return err
			}
			if len(args) == 1 && args[0] != id.ID {
				return fmt.Errorf("identity is %q, asked for %q", id.ID, args[0])
			}
			return slaveLoop(cmd.Context(), app, id)
		},
	}
}

// slaveLoop keeps the bidirectional stream alive; the master replays the full
// spec on every (re)connect, so no change is ever missed.
func slaveLoop(ctx context.Context, app *entity.App, id *Identity) error {
	log := app.Log.With("node", id.ID, "grpc", id.GRPC)
	counter := newReconcileCounter()
	for attempt := 0; ; attempt++ {
		if err := watchOnce(ctx, log, id, counter); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			backoff := time.Duration(1<<min(attempt, maxBackoffShift)) * time.Second
			log.WarnContext(ctx, "watch failed", "error", err.Error(), "retry_in", backoff.String())
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func watchOnce(ctx context.Context, log *slog.Logger, id *Identity, counter metric.Int64Counter) error {
	conn, err := rpc.Dial(ctx, id.GRPC)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	stream, err := rpc.NewControlClient(conn).Watch(ctx)
	if err != nil {
		return fmt.Errorf("watch rpc: %w", err)
	}
	if err := stream.Send(&rpc.StreamMsg{Node: id.ID}); err != nil {
		return fmt.Errorf("hello: %w", err)
	}
	log.InfoContext(ctx, "watch connected")
	client := rpc.NewControlClient(conn)
	reconciler := doorman.NewReconciler(id.ID)
	var engines []entity.Engine
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			_, _ = client.ReportStatus(ctx, &rpc.Status{
				Node: id.ID, State: stateError, Detail: detailDead + err.Error(),
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
				log.ErrorContext(ctx, "reconcile failed", "error", err.Error())
				_, _ = client.ReportStatus(ctx, &rpc.Status{Node: id.ID, State: state, Detail: err.Error()})
				continue
			}
			if detail == "" {
				detail = detailSync
			}
			_, _ = client.ReportStatus(ctx, &rpc.Status{Node: id.ID, State: state, Detail: detail})
			counter.Add(ctx, 1, metric.WithAttributes(attribute.String("node", id.ID)))
			log.InfoContext(ctx, "reconciled", "change", detail)
		case *rpc.StreamMsg_Query:
			res := executeQuery(ctx, id.ID, engines, p.Query)
			if err := stream.Send(&rpc.StreamMsg{Node: id.ID, Payload: &rpc.StreamMsg_Result{Result: res}}); err != nil {
				return err
			}
		}
	}
}

// executeQuery runs a query in the engine's container, matched by name
// against the node's last desired slice.
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
	return &rpc.QueryResult{Id: q.Id, Error: errors.New("no engine named " + q.Engine).Error()}
}

func newReconcileCounter() metric.Int64Counter {
	c, _ := otel.Meter("noxis").Int64Counter(reconcileMetric,
		metric.WithDescription("reconciliations performed per node"))
	return c
}
