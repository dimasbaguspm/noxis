// Package slave is the worker daemon. `register slave` captures the node
// identity once; `run slave` reads it and needs no env at all. It binds the
// node's docker network, runs its SQL instances, and answers database ops.
package slave

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
	"noxis/internal/repositories/kafka"
	"noxis/internal/repositories/master"
	reposlave "noxis/internal/repositories/slave"
)

const (
	heartbeatInterval   = 15 * time.Second
	consumerGroupPrefix = "noxis." // distinct from topic prefix; stable groups keep offsets
)

// opReq crosses the wire to the slave: which doorman responsibility to run.
type opReq struct {
	Op     string `json:"op"` // query | alter | rollback | analytics
	Engine string `json:"engine"`
	SQL    string `json:"sql,omitempty"`
	Table  string `json:"table,omitempty"`
}

type opRes struct {
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
			id, err := reposlave.Load()
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

// slaveLoop consumes this node's topics and drives its runtime. The broker
// consumer is durable and reconnects internally; a restart resumes from the
// last committed offset — no reconnect/backoff loop.
func slaveLoop(ctx context.Context, base *slog.Logger, id *reposlave.Identity) error {
	log := base.With("node", id.ID, "broker", id.Broker)
	addrs := strings.Split(id.Broker, ",")
	topics := kafka.Topics{Prefix: id.TopicPrefix}

	cons, err := kafka.Consume(ctx, addrs, consumerGroupPrefix+id.ID, false,
		topics.Spec(id.ID), topics.Req(id.ID))
	if err != nil {
		return err
	}
	defer cons.Close()

	producer, err := kafka.Produce(ctx, addrs)
	if err != nil {
		return err
	}
	defer producer.Close()
	if err := producer.EnsureTopic(ctx, topics.Status()); err != nil {
		return err
	}

	nets := NewNetworkManager(id.ID)
	dm := doorman.New(nil)
	var lastSpec entity.Spec

	// Startup refetch: durable offsets resume mid-stream, so a restart never
	// sees its last spec again. The master holds the current spec in memory —
	// ask it directly (identity carries the master URL; still zero env).
	replayed, err := master.NewClient(id.Master).GetSpec(ctx, id.ID)
	if err != nil {
		log.WarnContext(ctx, "spec refetch skipped", "error", err.Error())
	} else {
		log.InfoContext(ctx, "spec replayed from master", "engines", len(replayed.Engines))
		body, _ := json.Marshal(replayed)
		lastSpec = ackSpec(ctx, log, producer, topics, nets, dm, id.ID, body)
	}

	recCh := make(chan kafka.Record)
	go feed(ctx, cons, recCh)

	publishStatus(ctx, producer, topics, id.ID, "up", "slave online")
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	log.InfoContext(ctx, "slave consuming")

	for {
		select {
		case rec, ok := <-recCh:
			if !ok {
				return nil
			}
			switch rec.Topic {
			case topics.Spec(id.ID):
				lastSpec = ackSpec(ctx, log, producer, topics, nets, dm, id.ID, rec.Value)
			case topics.Req(id.ID):
				answerOp(ctx, log, producer, topics, dm, id.ID, lastSpec, rec)
			}
		case <-ticker.C:
			publishStatus(ctx, producer, topics, id.ID, "up", "slave online")
		case <-ctx.Done():
			return nil
		}
	}
}

// feed forwards consumed records until ctx dies; cons reconnects internally.
func feed(ctx context.Context, cons *kafka.Broker, out chan<- kafka.Record) {
	defer close(out)
	for {
		rec, err := cons.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		select {
		case out <- rec:
		case <-ctx.Done():
			return
		}
	}
}

// ackSpec converges the node on the desired state: network boundary first,
// then one instance per assigned engine. Reports each step in status.
func ackSpec(ctx context.Context, log *slog.Logger, producer *kafka.Broker, topics kafka.Topics,
	nets *NetworkManager, dm *doorman.Doorman, node string, body []byte) entity.Spec {
	var spec entity.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		log.WarnContext(ctx, "bad spec event", "error", err)
		return spec
	}
	var details []string
	state := "ok"
	netDetail, err := nets.Reconcile(ctx, spec.Engines)
	if err != nil {
		state = "error"
		netDetail = "network error: " + err.Error()
	}
	if netDetail != "" {
		details = append(details, netDetail)
	}
	for _, eng := range spec.Engines {
		d, err := dm.CreateInstance(ctx, node, nets.NetworkName(), eng)
		if err != nil {
			state = "error"
			d = fmt.Sprintf("instance %s error: %s", eng.Name, err.Error())
		}
		details = append(details, d)
	}
	detail := strings.Join(details, "; ")
	publishStatus(ctx, producer, topics, node, state, detail)
	log.InfoContext(ctx, "spec reconciled", "detail", detail)
	return spec
}

func publishStatus(ctx context.Context, producer *kafka.Broker, topics kafka.Topics, node, state, detail string) {
	body, err := json.Marshal(entity.NodeStatus{
		Node: node, State: state, Detail: detail, TS: time.Now().Unix(),
	})
	if err != nil {
		return
	}
	_ = producer.PublishSync(ctx, topics.Status(), node, body)
}

// answerOp runs one doorman responsibility and replies keyed by the request
// id (master correlates).
func answerOp(ctx context.Context, log *slog.Logger, producer *kafka.Broker, topics kafka.Topics,
	dm *doorman.Doorman, node string, spec entity.Spec, rec kafka.Record) {
	var req opReq
	if err := json.Unmarshal(rec.Value, &req); err != nil {
		return
	}
	log.InfoContext(ctx, "op received", "op", req.Op, "engine", req.Engine, "key", rec.Key)
	res := opRes{}
	eng, ok := engineOf(spec, req.Engine)
	if !ok {
		res.Error = "no engine named " + req.Engine + " on this node"
	} else {
		res = runOp(ctx, dm, node, req, eng)
	}
	body, err := json.Marshal(res)
	if err != nil {
		return
	}
	if err := producer.PublishSync(ctx, topics.Res(node), rec.Key, body); err != nil {
		log.WarnContext(ctx, "publish op answer", "error", err)
	}
	log.InfoContext(ctx, "op answered", "op", req.Op, "error", res.Error != "")
}

func runOp(ctx context.Context, dm *doorman.Doorman, node string, req opReq, eng entity.Engine) opRes {
	var (
		out string
		err error
	)
	switch req.Op {
	case "query":
		out, err = dm.Query(ctx, node, eng, req.SQL)
	case "alter":
		out, err = dm.AlterTable(ctx, node, eng, req.SQL)
	case "rollback":
		out, err = dm.RollbackTable(ctx, node, eng, req.SQL)
	case "analytics":
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
