// Package kafka is the event transport (franz-go). Master publishes
// desired-state events; slaves consume their own topics via a durable
// consumer group (offsets survive restarts, so no reconnect/replay loop is
// needed). Broker-agnostic over the Kafka protocol — swap the compose broker
// image and nothing else changes.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Topics names events by prefix (NOXIS_TOPIC_PREFIX), so independent
// environments can share one broker without colliding.
type Topics struct{ Prefix string }

func (t Topics) Status() string          { return t.Prefix + "status" }
func (t Topics) Spec(node string) string { return t.Prefix + "specs." + node }
func (t Topics) Req(node string) string  { return t.Prefix + "query.req." + node }
func (t Topics) Res(node string) string  { return t.Prefix + "query.res." + node }

const retries = 5

// Broker wraps a franz-go client for producing and consuming events.
type Broker struct {
	cl      *kgo.Client
	addrs   []string
	recvBuf atomic.Int64 // fetches received (debug/metrics hook)
}

// Produce opens a client for publishing events (one per process side).
func Produce(ctx context.Context, addrs []string) (*Broker, error) {
	cl, err := kgo.NewClient(kgo.SeedBrokers(addrs...),
		kgo.ProducerBatchCompression(kgo.NoCompression()))
	if err != nil {
		return nil, fmt.Errorf("broker: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := cl.Ping(ctx); err != nil {
		cl.Close()
		return nil, fmt.Errorf("broker ping: %w", err)
	}
	b, err := kadm.NewClient(cl).ListBrokers(ctx)
	if err != nil {
		slog.Warn("broker metadata error", "error", err.Error())
	} else {
		for _, l := range b {
			slog.Info("broker metadata", "id", l.NodeID, "host", l.Host, "port", l.Port)
		}
	}
	return &Broker{cl: cl, addrs: addrs}, nil
}

// PublishSync writes one event with bounded retries; a transient broker error
// (warm-up, rebalance) is retried instead of failing the caller's request.
func (b *Broker) PublishSync(ctx context.Context, topic, key string, value []byte) error {
	var last error
	for i := 0; i < retries; i++ {
		res := b.cl.ProduceSync(ctx, &kgo.Record{Topic: topic, Key: []byte(key), Value: value})
		if err := res.FirstErr(); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return err
		} else {
			last = err
		}
		time.Sleep(time.Duration(i+1) * 300 * time.Millisecond)
	}
	return last
}

// Close shuts down the underlying client.
func (b *Broker) Close() {
	if b.cl != nil {
		b.cl.Close()
	}
}

// EnsureTopic creates the topic if missing (idempotent). Called before any
// produce/consume so zero broker configuration is required. Uses its own
// client: closing a kadm admin that wraps the shared client would kill it.
func (b *Broker) EnsureTopic(ctx context.Context, topic string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cl, err := kgo.NewClient(kgo.SeedBrokers(b.addrs...))
	if err != nil {
		return err
	}
	defer cl.Close()
	adm := kadm.NewClient(cl)
	defer adm.Close()
	var last error
	for i := 0; i < retries; i++ {
		_, err := adm.CreateTopic(ctx, 1, 1, nil, topic)
		if err == nil || errors.Is(err, kerr.TopicAlreadyExists) {
			return nil
		}
		if i == 0 {
			slog.Warn("ensure topic failed", "topic", topic, "error", err.Error())
		}
		last = err
		if ctx.Err() != nil {
			return err
		}
		time.Sleep(time.Duration(i+1) * 300 * time.Millisecond)
	}
	return last
}

// Consume opens a consumer. Grouped consumers persist offsets (durable
// resume — committed offsets win; a fresh group starts at the start, so a
// brand-new slave replays its spec history and converges to the last one;
// replayed request records are answered into the void, harmless). A
// standalone fromEnd consumer watches only new records; a standalone
// non-fromEnd consumer scrolls from the start (reply matching by key makes
// replay harmless).
func Consume(ctx context.Context, addrs []string, group string, fromEnd bool, topics ...string) (*Broker, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(addrs...),
		kgo.ConsumeTopics(topics...),
		// default batch sizes are fine for control-plane payloads
	}
	switch {
	case group != "":
		opts = append(opts, kgo.ConsumerGroup(group), kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	case fromEnd:
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	default:
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	}
	cl, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &Broker{cl: cl, addrs: addrs}, nil
}

// Record is one consumed event.
type Record struct {
	Topic string
	Key   string
	Value []byte
}

// Next blocks for the next fetched record and returns its error, if any.
func (b *Broker) Next(ctx context.Context) (Record, error) {
	for {
		fetches := b.cl.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			return Record{}, errs[0].Err
		}
		for _, r := range fetches.Records() {
			b.recvBuf.Add(1)
			return Record{Topic: r.Topic, Key: string(r.Key), Value: r.Value}, nil
		}
		if ctx.Err() != nil {
			return Record{}, ctx.Err()
		}
	}
}

// Prime forces the consumer to establish its fetch session/position: one
// poll, result discarded. Used before publishing a request-reply so the
// reply record cannot race the consumer's first fetch.
func (b *Broker) Prime(ctx context.Context) error {
	_, err := b.Next(ctx)
	return err
}

// WaitKey consumes until a record with the given key arrives or the context
// dies. Used for query reply correlation (one throwaway consumer per query).
func (b *Broker) WaitKey(ctx context.Context, key string) ([]byte, error) {
	for {
		rec, err := b.Next(ctx)
		if err != nil {
			return nil, err
		}
		if rec.Key == key {
			return rec.Value, nil
		}
	}
}

// Log is a small helper for consumer loops to report fetch errors once.
func (b *Broker) Log(log *slog.Logger, err error) {
	log.Warn("broker fetch error", "error", err.Error())
}
