package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/services/mongodb"
)

// The MongoDB operations. Connection CRUD and the change routes are shared
// generically with the other families; what is MongoDB-only is the
// database/collection explorer and the read request runner.
//
// Requests run directly: a document change has no generic rollback, so the
// guards are the op allowlist and the refusal to update or delete without a
// filter (an empty filter would touch every document).

// @Summary List databases with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Success 200 {object} object{databases=[]mongodb.Database}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/databases [get]
func ListDatabases(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		m, err := d.engineMongo(r)
		if err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, m, r.PathValue("name"), entity.LevelRead)
		if err != nil {
			return nil, err
		}
		databases, err := m.Databases(ctx, *c)
		return map[string]any{"databases": databases}, err
	})
}

// @Summary List a database's collections with their sizes
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} object{collections=[]mongodb.Collection}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/collections [get]
func ListCollections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		m, err := d.engineMongo(r)
		if err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, m, r.PathValue("name"), entity.LevelRead)
		if err != nil {
			return nil, err
		}
		collections, err := m.Collections(ctx, *c, r.URL.Query().Get("database"))
		return map[string]any{"collections": collections}, err
	})
}

// @Summary Run a read request (find | aggregate | count)
// @Param engine path string true "engine family (mongodb)"
// @Param name path string true "connection name"
// @Param body body mongodb.Request true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} mongodb.Result
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /v1/engine/{engine}/connections/{name}/requests [post]
func PostRequest(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var req interfaces.MongoRequest
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		m, err := d.engineMongo(r)
		if err != nil {
			return nil, err
		}
		// a write op is refused by Run, but it must still not be handed to a
		// read-only caller
		level := entity.LevelRead
		if mongodb.IsWrite(req.Op) {
			level = entity.LevelWrite
		}
		c, err := d.raw(ctx, m, r.PathValue("name"), level)
		if err != nil {
			return nil, err
		}
		return m.Run(ctx, *c, req)
	})
}

// mongoChangeBody is one submitted request: the connection and notes of a
// change, plus the request itself, flat so the console sends one object.
type mongoChangeBody struct {
	Connection string `json:"connection"`
	Notes      string `json:"notes"`
	interfaces.MongoRequest
}

// submitMongoChange validates a write request on a scratch collection and
// records it for review. Like the SQL flow, a request that fails validation is
// returned with its checks and never recorded.
func (d Deps) submitMongoChange(ctx context.Context, m interfaces.MongoModule, body mongoChangeBody) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	conn, err := d.raw(ctx, m, body.Connection, entity.LevelWrite)
	if err != nil {
		return entity.Change{}, err
	}
	req := body.MongoRequest
	ch := entity.Change{
		ID:         idgen.UUID(),
		Connection: body.Connection,
		Notes:      body.Notes,
		Kind:       constants.KindMongo,
		Status:     constants.ChangeValidated,
		CreatedAt:  time.Now().Unix(),
		Rollback:   "",
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return entity.Change{}, err
	}
	// the request rides in Query as JSON: the console parses it back into the
	// filter and value editors.
	ch.Query = string(encoded)
	target := req.Database + "." + req.Collection
	ch.AffectedTables = []string{target}

	kind := entity.ChangeCheck{Target: "op kind"}
	if !mongodb.IsWrite(req.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in Explore", req.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, req.Op+" on "+target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (scratch collection)"}
	if res, err := m.ValidateRequest(ctx, *conn, req); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
	} else {
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)

	if ch.Status == constants.ChangeFailed {
		ch.ID = ""
		return ch, nil
	}
	if err := d.Changes.SaveChange(ctx, ch); err != nil {
		return entity.Change{}, err
	}
	return ch, nil
}

// mongoChange loads a change and insists it is a MongoDB request, so the SQL
// queue cannot be addressed through the MongoDB flow.
func (d Deps) mongoChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindMongo {
		return nil, fmt.Errorf("change %s is %s, not a mongodb request", id, ch.Kind)
	}
	if err := d.authorize(ctx, ch.Connection, entity.LevelRead); err != nil {
		return nil, err
	}
	return ch, nil
}

// applyMongo executes an approved request; a failure is recorded on the
// change, and there is no rollback to fall back on.
func (d Deps) applyMongo(ctx context.Context, m interfaces.MongoModule, id string) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	ch, err := d.mongoChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("request is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := d.raw(ctx, m, ch.Connection, entity.LevelAdmin)
	if err != nil {
		return entity.Change{}, err
	}
	var req interfaces.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Change{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	res, err := m.Apply(ctx, *conn, req)
	if err != nil {
		ch.Status = constants.ChangeFailed
		ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", Error: err.Error()})
		_ = d.Changes.SaveChange(ctx, *ch)
		return *ch, err
	}
	ch.Status, ch.AppliedAt = constants.ChangeApplied, time.Now().Unix()
	ch.AppliedAffected, ch.AppliedDurationMs = res.Affected, res.DurationMs
	ch.Checks = append(ch.Checks, entity.ChangeCheck{Target: "apply", OK: true, Detail: res.Message})
	if err := d.Changes.SaveChange(ctx, *ch); err != nil {
		return entity.Change{}, err
	}
	return *ch, nil
}

// mongoChanges lists only MongoDB requests: each family keeps its own queue.
func (d Deps) mongoChanges(ctx context.Context) (out []entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	all, err := d.Changes.ListChanges(ctx)
	if err != nil {
		return nil, err
	}
	readable, err := d.readableConnections(ctx)
	if err != nil {
		return nil, err
	}
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Kind == constants.KindMongo && readable[all[i].Connection] {
			out = append(out, all[i])
		}
	}
	return out, nil
}
