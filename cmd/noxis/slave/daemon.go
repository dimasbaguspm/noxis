// Package slave is the worker daemon. `register slave` captures the node
// identity once; `run slave` reads it and needs no env at all.
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
	"noxis/internal/features/doorman"
	"noxis/internal/repositories/kafka"
	reposlave "noxis/internal/repositories/slave"
)

const heartbeatInterval = 15 * time.Second

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

// slaveLoop consumes this node's topics and reports status. There is no
// reconnect/backoff loop: the broker consumer is durable and reconnects
// internally; a restart resumes from the last committed offset. Heartbeats
// keep the master's status view fresh (heartbeatInterval).
func slaveLoop(ctx context.Context, base *slog.Logger, id *reposlave.Identity) error {
	log := base.With("node", id.ID, "broker", id.Broker)
	addrs := strings.Split(id.Broker, ",")

	topics := kafka.Topics{Prefix: id.TopicPrefix}
	cons, err := kafka.Consume(ctx, addrs, "noxis."+id.ID, false,
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

	nets := doorman.NewNetworkManager(id.ID)

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
				ackSpec(ctx, log, producer, topics, nets, id.ID, rec.Value)
			case topics.Req(id.ID):
				answerQuery(ctx, log, producer, topics, id.ID, rec)
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

// ackSpec acknowledges a desired-state push. Runtime convergence (the old
// doorman engine manager) is paused for the single-machine trial — the spec
// event round-trip itself is what matters now.
func ackSpec(ctx context.Context, log *slog.Logger, producer *kafka.Broker, topics kafka.Topics, nets *doorman.NetworkManager, node string, body []byte) {
	var spec entity.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		log.WarnContext(ctx, "bad spec event", "error", err)
		return
	}
	detail, err := nets.Reconcile(ctx, spec.Engines)
	state := "ok"
	if err != nil {
		state = "error"
		log.ErrorContext(ctx, "network reconcile failed", "error", err.Error())
		detail = "network error: " + err.Error()
	}
	publishStatus(ctx, producer, topics, node, state, detail)
	log.InfoContext(ctx, "spec reconciled", "detail", detail)
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

// answerQuery replies keyed by the request id (master correlates). No engine
// runtime is installed yet (doorman removed); queries answer with a clear
// error so the transport round-trip still validates.
func answerQuery(ctx context.Context, log *slog.Logger, producer *kafka.Broker, topics kafka.Topics, node string, rec kafka.Record) {
	var req queryReq
	if err := json.Unmarshal(rec.Value, &req); err != nil {
		return
	}
	res := queryRes{Error: "engine runtime not installed (doorman removed)"}
	body, err := json.Marshal(res)
	if err != nil {
		return
	}
	if err := producer.PublishSync(ctx, topics.Res(node), rec.Key, body); err != nil {
		log.WarnContext(ctx, "publish query answer", "error", err)
	}
}
