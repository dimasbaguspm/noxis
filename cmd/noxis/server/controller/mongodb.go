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

// The MongoDB namespace. Connection CRUD shares one implementation with the
// other families; what is MongoDB-only is the database/collection explorer,
// the CRUD request runner and the stats.
//
// Requests run directly: a document change has no generic rollback, so the
// guards are the op allowlist and the refusal to update or delete without a
// filter (an empty filter would touch every document).

// @Summary List MongoDB connections (passwords redacted)
// @Success 200 {object} object{connections=[]entity.Connection}
// @Router /v1/mongodb/connections [get]
func ListMongoConnections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		conns, err := d.connections(ctx, d.Mongo)
		return map[string]any{"connections": conns}, err
	})
}

// @Summary Create or update a MongoDB connection
// @Param body body entity.Connection true "connection (omit password to keep the stored one)"
// @Success 200 {object} entity.Connection
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections [post]
func SaveMongoConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var c entity.Connection
		if err := decode(r, &c); err != nil {
			return nil, err
		}
		return d.saveConnection(ctx, d.Mongo, c)
	})
}

// @Summary Get one MongoDB connection (password redacted)
// @Param name path string true "connection name"
// @Success 200 {object} entity.Connection
// @Failure 404 {object} map[string]string
// @Router /v1/mongodb/connections/{name} [get]
func GetMongoConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.connection(ctx, d.Mongo, r.PathValue("name"))
	})
}

// @Summary Delete a MongoDB connection
// @Param name path string true "connection name"
// @Success 204 "deleted"
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name} [delete]
func DeleteMongoConnection(w http.ResponseWriter, r *http.Request) {
	handleNoContent(w, r, func(ctx context.Context, d Deps) error {
		return d.deleteConnection(ctx, d.Mongo, r.PathValue("name"))
	})
}

// @Summary Test that a MongoDB connection answers
// @Param name path string true "connection name"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/test [post]
func TestMongoConnection(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if err := d.testConnection(ctx, d.Mongo, r.PathValue("name")); err != nil {
			return nil, err
		}
		return map[string]string{"status": "ok"}, nil
	})
}

// @Summary List databases with their sizes
// @Param name path string true "connection name"
// @Success 200 {object} object{databases=[]mongodb.Database}
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/databases [get]
func ListMongoDatabases(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		databases, err := d.Mongo.Databases(ctx, *c)
		return map[string]any{"databases": databases}, err
	})
}

// @Summary List a database's collections with their sizes
// @Param name path string true "connection name"
// @Param database query string true "database name"
// @Success 200 {object} object{collections=[]mongodb.Collection}
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/collections [get]
func ListMongoCollections(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		collections, err := d.Mongo.Collections(ctx, *c, r.URL.Query().Get("database"))
		return map[string]any{"collections": collections}, err
	})
}

// @Summary Run a read request (find | aggregate | count)
// @Param name path string true "connection name"
// @Param body body mongodb.Request true "op, database, collection, filter and value as Extended JSON"
// @Success 200 {object} mongodb.Result
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/requests [post]
func PostMongoRequest(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var req interfaces.MongoRequest
		if err := decode(r, &req); err != nil {
			return nil, err
		}
		c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
		if err != nil {
			return nil, err
		}
		return d.Mongo.Run(ctx, *c, req)
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
func (d Deps) submitMongoChange(ctx context.Context, body mongoChangeBody) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	conn, err := d.raw(ctx, d.Mongo, body.Connection)
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
	if res, err := d.Mongo.ValidateRequest(ctx, *conn, req); err == nil {
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
// queues cannot be addressed through this namespace.
func (d Deps) mongoChange(ctx context.Context, id string) (*entity.Change, error) {
	ch, err := d.Changes.GetChange(ctx, id)
	if err != nil {
		return nil, err
	}
	if ch.Kind != constants.KindMongo {
		return nil, fmt.Errorf("change %s is %s, not a mongodb request", id, ch.Kind)
	}
	return ch, nil
}

// applyMongo executes an approved request; a failure is recorded on the
// change, and there is no rollback to fall back on.
func (d Deps) applyMongo(ctx context.Context, id string) (out entity.Change, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	ch, err := d.mongoChange(ctx, id)
	if err != nil {
		return entity.Change{}, err
	}
	if ch.Status != constants.ChangeApproved {
		return *ch, fmt.Errorf("request is %s, not approved — cannot apply", ch.Status)
	}
	conn, err := d.raw(ctx, d.Mongo, ch.Connection)
	if err != nil {
		return entity.Change{}, err
	}
	var req interfaces.MongoRequest
	if err := json.Unmarshal([]byte(ch.Query), &req); err != nil {
		return entity.Change{}, fmt.Errorf("stored request is unreadable: %w", err)
	}
	res, err := d.Mongo.Apply(ctx, *conn, req)
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
	out = make([]entity.Change, 0, len(all))
	for i := range all {
		if all[i].Kind == constants.KindMongo {
			out = append(out, all[i])
		}
	}
	return out, nil
}

// @Summary List MongoDB requests (newest first)
// @Success 200 {object} object{changes=[]entity.Change}
// @Router /v1/mongodb/changes [get]
func ListMongoChanges(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		changes, err := d.mongoChanges(ctx)
		return map[string]any{"changes": changes}, err
	})
}

// @Summary Submit a MongoDB request (validated on a scratch collection)
// @Param body body controller.mongoChangeBody true "connection, notes and the request"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/changes [post]
func SubmitMongoChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		var body mongoChangeBody
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.submitMongoChange(ctx, body)
	})
}

// @Summary Get one MongoDB request
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 404 {object} map[string]string
// @Router /v1/mongodb/changes/{id} [get]
func GetMongoChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.mongoChange(ctx, r.PathValue("id"))
	})
}

// @Summary Approve a validated MongoDB request (self-approval allowed)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/changes/{id}/approve [post]
func ApproveMongoChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if _, err := d.mongoChange(ctx, r.PathValue("id")); err != nil {
			return nil, err
		}
		return d.approve(ctx, r.PathValue("id"))
	})
}

// @Summary Apply an approved MongoDB request (no rollback exists)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/changes/{id}/apply [post]
func ApplyMongoChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.applyMongo(ctx, r.PathValue("id"))
	})
}

// @Summary Reject a MongoDB request (kept as an artifact, never deleted)
// @Param id path string true "change id"
// @Success 200 {object} entity.Change
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/changes/{id}/reject [post]
func RejectMongoChange(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		if _, err := d.mongoChange(ctx, r.PathValue("id")); err != nil {
			return nil, err
		}
		return d.reject(ctx, r.PathValue("id"))
	})
}

// @Summary MongoDB stats snapshot (server, database and collection)
// @Param name path string true "connection name"
// @Param database query string false "database to describe"
// @Param collection query string false "collection to describe"
// @Success 200 {object} mongodb.Stats
// @Failure 400 {object} map[string]string
// @Router /v1/mongodb/connections/{name}/stats [get]
func GetMongoStats(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		return d.mongoStats(ctx, r)
	})
}

// @Summary MongoDB stats stream (SSE, one snapshot every 2s)
// @Param name path string true "connection name"
// @Param database query string false "database to describe"
// @Param collection query string false "collection to describe"
// @Success 200 {string} string "text/event-stream"
// @Router /v1/mongodb/connections/{name}/stats/stream [get]
func StreamMongoStats(w http.ResponseWriter, r *http.Request) {
	d := From(r)
	streamSnapshots(w, r, func(ctx context.Context) (any, error) { return d.mongoStats(ctx, r) })
}

func (d Deps) mongoStats(ctx context.Context, r *http.Request) (out any, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	c, err := d.raw(ctx, d.Mongo, r.PathValue("name"))
	if err != nil {
		return nil, err
	}
	return d.Mongo.Stats(ctx, *c, r.URL.Query().Get("database"), r.URL.Query().Get("collection"))
}
