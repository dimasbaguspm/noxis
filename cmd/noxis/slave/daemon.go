package slave

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"noxis/internal/entity"
	"noxis/internal/events"
	"noxis/internal/features/doorman"
	"noxis/internal/features/doorman/runtimes"
)

const reconcileMetric = "noxis.reconcile.total"

type queryReq struct {
	Engine string `json:"engine"`
	SQL    string `json:"sql"`
}

type queryRes struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// NewRunCmd is `noxis run slave <id>`: starts the worker daemon from the
// identity file written by register. The id must match the identity.
func NewRunCmd(log *slog.Logger) *cobra.Command {
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
			return slaveLoop(cmd.Context(), log, id)
		},
	}
}

// slaveLoop consumes this node's spec + query topics and acts. There is no
// reconnect/backoff loop: the broker consumer is durable and reconnects
// internally; a restart resumes from the last committed offset.
func slaveLoop(ctx context.Context, base *slog.Logger, id *Identity) error {
	log := base.With("node", id.ID, "broker", id.Broker)
	cons, err := events.Consume(ctx, strings.Split(id.Broker, ","), "noxis."+id.ID, false,
		events.SpecTopic(id.ID), events.ReqTopic(id.ID))
	if err != nil {
		return err
	}
	defer cons.Close()

	producer, err := events.Produce(ctx, strings.Split(id.Broker, ","))
	if err != nil {
		return err
	}
	defer producer.Close()

	reconciler := doorman.NewReconciler(id.ID)
	counter := newReconcileCounter()
	var lastSpec entity.Spec // for query engine resolution
	log.InfoContext(ctx, "slave consuming")

	for {
		rec, err := cons.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			cons.Log(log, err) // consumer reconnects internally
			continue
		}
		switch rec.Topic {
		case events.SpecTopic(id.ID):
			lastSpec = reconcile(ctx, log, producer, id.ID, reconciler, counter, rec.Value)
		case events.ReqTopic(id.ID):
			answerQuery(ctx, log, producer, id.ID, lastSpec, rec)
		}
	}
}

func reconcile(ctx context.Context, log *slog.Logger, producer *events.Broker, node string,
	reconciler *doorman.Reconciler, counter metric.Int64Counter, body []byte) entity.Spec {
	var spec entity.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		log.WarnContext(ctx, "bad spec event", "error", err)
		return spec
	}
	detail, err := reconciler.Reconcile(ctx, spec.Engines)
	state := "ok"
	if err != nil {
		state = "error"
		log.ErrorContext(ctx, "reconcile failed", "error", err.Error())
		detail = err.Error()
	} else if detail == "" {
		detail = "in sync"
	}
	publishStatus(ctx, producer, node, state, detail)
	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("node", node)))
	log.InfoContext(ctx, "reconciled", "change", detail)
	return spec
}

func publishStatus(ctx context.Context, producer *events.Broker, node, state, detail string) {
	body, err := json.Marshal(entity.NodeStatus{
		Node: node, State: state, Detail: detail, TS: time.Now().Unix(),
	})
	if err != nil {
		return
	}
	if err := producer.EnsureTopic(ctx, events.TopicStatus); err != nil {
		return
	}
	_ = producer.PublishSync(ctx, events.TopicStatus, node, body)
}

// answerQuery runs the SQL in the matching engine container and publishes the
// reply keyed by the request id (master correlates). Engines resolve against
// the last desired spec this node reconciled.
func answerQuery(ctx context.Context, log *slog.Logger, producer *events.Broker, node string, spec entity.Spec, rec events.Record) {
	var req queryReq
	if err := json.Unmarshal(rec.Value, &req); err != nil {
		return
	}
	res := queryRes{}
	matched := false
	for _, e := range spec.Engines {
		if e.Name != req.Engine {
			continue
		}
		matched = true
		out, err := runtimes.ExecSQL(ctx, runtimes.ContainerName(node, e.Name), e, req.SQL)
		if err != nil {
			res.Error = err.Error()
		} else {
			res.Output = out
		}
	}
	if !matched {
		res.Error = "no engine named " + req.Engine + " on this node"
	}
	body, err := json.Marshal(res)
	if err != nil {
		return
	}
	if err := producer.PublishSync(ctx, events.ResTopic(node), rec.Key, body); err != nil {
		log.WarnContext(ctx, "publish query answer", "error", err)
	}
}

func newReconcileCounter() metric.Int64Counter {
	c, _ := otel.Meter("noxis").Int64Counter(reconcileMetric,
		metric.WithDescription("reconciliations performed per node"))
	return c
}
