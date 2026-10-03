// Package slave is the worker daemon module. `register slave` captures the
// node identity once; `run slave` reads it and needs no env at all. Run is
// the module entry: it binds the node's docker network, keeps its SQL
// instances on the desired spec, and answers database ops (handlers).
package slave

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"noxis/cmd/noxis/slave/handlers"
	"noxis/internal/entity"
	"noxis/internal/modules/doorman"
	"noxis/internal/repositories/docker"
	"noxis/internal/repositories/kafka"
	"noxis/internal/repositories/master"
	reposlave "noxis/internal/repositories/slave"
)

const (
	heartbeatInterval   = 15 * time.Second
	consumerGroupPrefix = "noxis." // distinct from topic prefix; stable groups keep offsets
)

// Run is the module entry: the worker daemon for one node.
func Run(ctx context.Context, base *slog.Logger, id *reposlave.Identity) error {
	return slaveLoop(ctx, base, id)
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

	// One docker client drives both the node network and the instances.
	d := docker.New()
	dm := doorman.New(d)
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
		lastSpec = ackSpec(ctx, log, producer, topics, d, dm, id.ID, body)
	}

	deps := handlers.Deps{Log: log, Producer: producer, Topics: topics, Dm: dm}

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
				lastSpec = ackSpec(ctx, log, producer, topics, d, dm, id.ID, rec.Value)
			case topics.Req(id.ID):
				deps.Answer(ctx, id.ID, lastSpec, rec)
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
	d *docker.Client, dm *doorman.Doorman, node string, body []byte) entity.Spec {
	var spec entity.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		log.WarnContext(ctx, "bad spec event", "error", err)
		return spec
	}
	var details []string
	state := "ok"
	netDetail, err := d.NetworkReconcile(ctx, docker.NodeNetworkName(node), docker.NodeNetworkLabel(node), len(spec.Engines))
	if err != nil {
		state = "error"
		netDetail = "network error: " + err.Error()
	}
	if netDetail != "" {
		details = append(details, netDetail)
	}
	for _, eng := range spec.Engines {
		instanceDetail, err := dm.CreateInstance(ctx, node, docker.NodeNetworkName(node), eng)
		if err != nil {
			state = "error"
			instanceDetail = fmt.Sprintf("instance %s error: %s", eng.Name, err.Error())
		}
		details = append(details, instanceDetail)
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
