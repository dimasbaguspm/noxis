// Package control is the master's operations core: repositories + broker.
// REST handlers (cmd/noxis/master/controller) call it; slaves never do.
package control

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"noxis/internal/entity"
	"noxis/internal/events"
	"noxis/internal/repository"
)

const queryTimeout = 30 * time.Second

// Control owns the master's data + event sides.
type Control struct {
	repo  *repository.SQLite
	k     *events.Broker
	log   *slog.Logger
	addrs []string
}

func New(repo *repository.SQLite, k *events.Broker, log *slog.Logger, addrs []string) *Control {
	return &Control{repo: repo, k: k, log: log, addrs: addrs}
}

func (c *Control) Spec(ctx context.Context, node string) (*entity.Spec, error) {
	return c.repo.Get(ctx, node)
}

// SetSpec persists the assignment then publishes it to the node's topic.
// A crash between the two self-heals: the next publish carries the full spec.
func (c *Control) SetSpec(ctx context.Context, spec *entity.Spec) error {
	if err := c.repo.Set(ctx, spec); err != nil {
		return err
	}
	if err := c.k.EnsureTopic(ctx, events.SpecTopic(spec.Node)); err != nil {
		return err
	}
	body, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return c.k.PublishSync(ctx, events.SpecTopic(spec.Node), spec.Node, body)
}

func (c *Control) RegisterNode(ctx context.Context, id string) error {
	return c.repo.Register(ctx, id)
}

// Registered returns the registered node ids (for status merge).
func (c *Control) Registered(ctx context.Context) ([]string, error) {
	return c.repo.ListIDs(ctx)
}

func (c *Control) Statuses(ctx context.Context) ([]entity.NodeStatus, error) {
	return c.repo.List(ctx)
}

// RunStatusConsumer drains the status topic into the repo until ctx is done.
func (c *Control) RunStatusConsumer(ctx context.Context) error {
	if err := c.k.EnsureTopic(ctx, events.TopicStatus); err != nil {
		return err
	}
	cons, err := events.Consume(ctx, c.addrs, "", true, events.TopicStatus)
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
		if err := c.repo.Report(ctx, st); err != nil {
			c.log.WarnContext(ctx, "report status failed", "error", err)
		}
	}
}

type queryReq struct {
	Engine string `json:"engine"`
	SQL    string `json:"sql"`
}

type queryRes struct {
	Output string `json:"output"`
	Error  string `json:"error"`
}

// Query publishes a request to the node's topic and waits for the correlated
// reply. One throwaway consumer per query keeps reply routing trivial.
//
// ponytail: per-query consumer connection; fine at personal-infra query rates
// — reuse a pool if queries ever become frequent.
func (c *Control) Query(ctx context.Context, node, engine, sql string) (string, error) {
	qid, err := newID()
	if err != nil {
		return "", err
	}
	if err := c.k.EnsureTopic(ctx, events.ReqTopic(node)); err != nil {
		return "", err
	}
	if err := c.k.EnsureTopic(ctx, events.ResTopic(node)); err != nil {
		return "", err
	}
	req, err := json.Marshal(queryReq{Engine: engine, SQL: sql})
	if err != nil {
		return "", err
	}
	if err := c.k.PublishSync(ctx, events.ReqTopic(node), qid, req); err != nil {
		return "", err
	}

	cons, err := events.Consume(ctx, c.addrs, "", true, events.ResTopic(node))
	if err != nil {
		return "", err
	}
	defer cons.Close()
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	body, err := cons.WaitKey(ctx, qid)
	if err != nil {
		return "", errors.New("query timed out: node " + node + " did not answer")
	}
	var res queryRes
	if err := json.Unmarshal(body, &res); err != nil {
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
