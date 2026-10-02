// Package control is the master daemon's operations core: registry repo +
// Kafka broker + memory state. Lives under cmd/noxis because it is command
// scope — used by the master daemon (server) and its REST handlers
// (controller), nothing in internal/ shares it. Slaves never touch it.
package control

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/repositories/kafka"
)

const (
	queryTimeout       = 30 * time.Second
	statusOfflineAfter = 30 * time.Second // no heartbeat in this window => offline
)

// Control owns the master's data + event sides. Desired specs and reported
// statuses live in memory; the sqlite store holds only the slave registry.
//
// ponytail: master restart loses specs/status until the slave's next heartbeat;
// if that ever matters, replay the spec topics on startup.
type Control struct {
	nodes  interfaces.NodeRepository
	topics kafka.Topics
	k      *kafka.Broker
	log    *slog.Logger
	addrs  []string

	mu     sync.Mutex
	specs  map[string]entity.Spec
	status map[string]entity.NodeStatus
}

func New(nodes interfaces.NodeRepository, topics kafka.Topics, k *kafka.Broker, log *slog.Logger, addrs []string) *Control {
	return &Control{
		nodes: nodes, topics: topics, k: k, log: log, addrs: addrs,
		specs: map[string]entity.Spec{}, status: map[string]entity.NodeStatus{},
	}
}

func (c *Control) Spec(ctx context.Context, node string) (*entity.Spec, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.specs[node]
	if !ok {
		return nil, errors.New("no spec assigned to node " + node)
	}
	return &s, nil
}

// SetSpec remembers the assignment then publishes it to the node's topic.
// A crash between the two self-heals: the next publish carries the full spec.
func (c *Control) SetSpec(ctx context.Context, spec *entity.Spec) error {
	if err := c.k.EnsureTopic(ctx, c.topics.Spec(spec.Node)); err != nil {
		return err
	}
	body, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err := c.k.PublishSync(ctx, c.topics.Spec(spec.Node), spec.Node, body); err != nil {
		return err
	}
	c.mu.Lock()
	c.specs[spec.Node] = *spec
	c.mu.Unlock()
	return nil
}

func (c *Control) RegisterNode(ctx context.Context, id, name, ip string) error {
	return c.nodes.Register(ctx, id, name, ip)
}

// Statuses merges the registered registry with live heartbeats: registered
// nodes with a stale (or missing) heartbeat read as offline.
func (c *Control) Statuses(ctx context.Context) ([]entity.NodeStatus, error) {
	slaves, err := c.nodes.List(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().Unix()
	out := make([]entity.NodeStatus, 0, len(slaves))
	for _, s := range slaves {
		st, ok := c.status[s.ID]
		if !ok || now-st.TS > int64(statusOfflineAfter.Seconds()) {
			out = append(out, entity.NodeStatus{Node: s.ID, State: "offline"})
			continue
		}
		out = append(out, st)
	}
	return out, nil
}

// RunStatusConsumer drains the status topic into memory until ctx is done.
func (c *Control) RunStatusConsumer(ctx context.Context) error {
	if err := c.k.EnsureTopic(ctx, c.topics.Status()); err != nil {
		return err
	}
	cons, err := kafka.Consume(ctx, c.addrs, "", true, c.topics.Status())
	if err != nil {
		return err
	}
	defer cons.Close()
	for {
		rec, err := cons.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			cons.Log(c.log, err)
			continue
		}
		var st entity.NodeStatus
		if err := json.Unmarshal(rec.Value, &st); err != nil {
			c.log.WarnContext(ctx, "bad status event", "node", rec.Key, "error", err)
			continue
		}
		c.mu.Lock()
		c.status[st.Node] = st
		c.mu.Unlock()
	}
}

// OpRequest is a doorman responsibility crossing to the slave over the
// node's request topic.
type OpRequest struct {
	Op     string `json:"op"` // query | alter | rollback | analytics
	Engine string `json:"engine"`
	SQL    string `json:"sql,omitempty"`
	Table  string `json:"table,omitempty"`
}

type opRes struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// Request publishes a doorman op to the node's topic and waits for the
// correlated reply. One throwaway consumer per request keeps reply routing
// trivial.
//
// ponytail: per-request consumer connection; fine at personal-infra rates —
// reuse a pool if requests ever become frequent.
func (c *Control) Request(ctx context.Context, node string, req OpRequest) (string, error) {
	qid, err := newID()
	if err != nil {
		return "", err
	}
	if err := c.k.EnsureTopic(ctx, c.topics.Req(node)); err != nil {
		return "", err
	}
	if err := c.k.EnsureTopic(ctx, c.topics.Res(node)); err != nil {
		return "", err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	if err := c.k.PublishSync(ctx, c.topics.Req(node), qid, body); err != nil {
		return "", err
	}

	// Scroll from the start of the topic so an answer published before this
	// consumer subscribed is never missed; stale replies are keyed by other
	// request ids and skipped. Subscribing first would also fix the race.
	cons, err := kafka.Consume(ctx, c.addrs, "", false, c.topics.Res(node))
	if err != nil {
		return "", err
	}
	defer cons.Close()
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	reply, err := cons.WaitKey(ctx, qid)
	if err != nil {
		return "", errors.New("request timed out: node " + node + " did not answer")
	}
	var res opRes
	if err := json.Unmarshal(reply, &res); err != nil {
		return "", err
	}
	if res.Error != "" {
		return "", errors.New(res.Error)
	}
	return res.Output, nil
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
