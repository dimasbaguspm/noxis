package change

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"noxis/internal/common/idgen"
	"noxis/internal/common/logging"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/usecases"
)

// Submit submits a change after validating it.
type Submit struct{ usecases.RegistryDeps }

// @Summary Submit a change (validates before recording)
// @Param driver path string true "driver name (postgres, mysql, mariadb, sqlite, cockroach, redis, valkey, mongo)"
// @Param body body entity.Change true "connection, notes, query, rollback (sql) or the request (mongodb)"
// @Success 200 {object} entity.Change
// @Failure 400 {object} entity.ErrorResponse
// @Failure 404 {object} entity.ErrorResponse
// @Router /external/v1/engine/{driver}/changes [post]
func (uc Submit) Serve(w http.ResponseWriter, r *http.Request) {
	ctx, end := logging.StartSpan(r.Context())
	defer end(nil)
	module, err := uc.Modules.Engine(r.PathValue("driver"))
	if err != nil {
		uc.Fail(ctx, w, err)
		return
	}
	switch engine := module.(type) {
	case interfaces.SQLChangeEngine:
		var change entity.Change
		if err := uc.Decode(r, &change); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		connection, err := uc.RawConn(ctx, engine, change.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := submitSQL(ctx, uc.RegistryDeps, engine, *connection, entity.Actor(ctx), change)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, module.Driver())
		uc.Respond(w, out)
	case interfaces.MongoChangeEngine:
		var body entity.MongoChangeRequest
		if err := uc.Decode(r, &body); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		connection, err := uc.RawConn(ctx, engine, body.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := submitDocument(ctx, uc.RegistryDeps, engine, *connection, entity.Actor(ctx), body)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, module.Driver())
		uc.Respond(w, out)
	case interfaces.SearchEngine:
		var body entity.SearchChangeRequest
		if err := uc.Decode(r, &body); err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		connection, err := uc.RawConn(ctx, engine, body.Connection, entity.LevelWrite)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		out, err := submitSearch(ctx, uc.RegistryDeps, engine, *connection, entity.Actor(ctx), body)
		if err != nil {
			uc.Fail(ctx, w, err)
			return
		}
		uc.Metrics.ChangeSubmitted(ctx, module.Driver())
		uc.Respond(w, out)
	default:
		uc.Fail(ctx, w, unsupportedChangeFlow(module))
	}
}

// submitSQL proves a batch and records it: the engine decides the kind and what it touches, and the dry run decides
// whether the change is validated, unverified or refused.
func submitSQL(ctx context.Context, deps usecases.RegistryDeps, engine interfaces.SQLChangeEngine, conn entity.Connection, actor string, ch entity.Change) (entity.Change, error) {
	if err := ch.Validate(); err != nil {
		return entity.Change{}, err
	}
	ch.ID, ch.CreatedAt, ch.CreatedBy = idgen.UUID(), time.Now().Unix(), actor
	derived := engine.Kind(ch.Query)
	if err := ch.RequireKind(derived); err != nil {
		return entity.Change{}, err
	}
	ch.Kind, ch.Status = derived, constants.ChangeValidated
	ch.AffectedTables, ch.AffectedColumns = engine.Referenced(ch.Query)
	ch.Checks = []entity.ChangeCheck{{Target: "query kind"}}
	if err := engine.VerifyWrite(ch.Query); err != nil {
		ch.Checks[0].Error = err.Error()
		ch.Status = constants.ChangeFailed
	} else {
		ch.Checks[0].OK = true
		ch.Checks[0].Detail = "write statements only"
	}
	dry := entity.ChangeCheck{Target: "dry-run (query + rollback)"}
	res, err := engine.ValidateChange(ctx, conn, ch.Query, ch.Rollback)
	switch {
	case err == nil:
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
		ch.ScratchSchema = res.ScratchSchema
	case errors.Is(err, interfaces.ErrUnverifiable):
		ch.Status, dry.Error = constants.ChangeUnverified, err.Error()
		ch.UnverifiedReason = interfaces.UnverifiableReason(err)
	default:
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)
	logSubmitted(ctx, ch, actor, dry.Error)
	return deps.RecordChange(ctx, ch)
}

// submitDocument proves a document request and records it.
func submitDocument(ctx context.Context, deps usecases.RegistryDeps, engine interfaces.MongoChangeEngine, conn entity.Connection, actor string, body entity.MongoChangeRequest) (entity.Change, error) {
	encoded, err := json.Marshal(body.MongoRequest)
	if err != nil {
		return entity.Change{}, err
	}
	target := body.Database + "." + body.Collection
	ch := entity.Change{
		ID: idgen.UUID(), Connection: conn.Name, Notes: body.Notes, Query: string(encoded),
		Kind: constants.KindMongo, Status: constants.ChangeValidated,
		CreatedAt: time.Now().Unix(), CreatedBy: actor, AffectedTables: []string{target},
	}

	kind := entity.ChangeCheck{Target: "op kind"}
	if !engine.IsWrite(body.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in Explore", body.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, body.Op+" on "+target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (scratch collection)"}
	if res, err := engine.ValidateRequest(ctx, conn, body.MongoRequest); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
		ch.ScratchSchema = res.ScratchSchema
	} else {
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)
	logSubmitted(ctx, ch, actor, dry.Error)
	return deps.RecordChange(ctx, ch)
}

// submitSearch proves a search-platform request and records it.
func submitSearch(ctx context.Context, deps usecases.RegistryDeps, engine interfaces.SearchEngine, conn entity.Connection, actor string, body entity.SearchChangeRequest) (entity.Change, error) {
	encoded, err := json.Marshal(body.SearchRequest)
	if err != nil {
		return entity.Change{}, err
	}
	ch := entity.Change{
		ID: idgen.UUID(), Connection: conn.Name, Notes: body.Notes, Query: string(encoded),
		Kind: constants.KindSearch, Status: constants.ChangeValidated,
		CreatedAt: time.Now().Unix(), CreatedBy: actor, AffectedTables: []string{body.Target},
	}

	kind := entity.ChangeCheck{Target: "op kind"}
	if !engine.IsWrite(body.Op) {
		kind.Error = fmt.Sprintf("op %q reads only: run it in the search page", body.Op)
		ch.Status = constants.ChangeFailed
	} else {
		kind.OK, kind.Detail = true, body.Op+" on "+body.Target
	}
	ch.Checks = append(ch.Checks, kind)

	dry := entity.ChangeCheck{Target: "dry-run (impact count)"}
	if res, err := engine.ValidateChange(ctx, conn, body.SearchRequest); err == nil {
		dry.OK, dry.Detail = true, res.Message
		ch.EstDurationMs, ch.EstAffected = res.DurationMs, res.Affected
	} else {
		ch.Status, dry.Error = constants.ChangeFailed, err.Error()
	}
	ch.Checks = append(ch.Checks, dry)
	logSubmitted(ctx, ch, actor, dry.Error)
	return deps.RecordChange(ctx, ch)
}

// logSubmitted records what the submission decided, so the audit trail says why a change was refused or recorded
// unverified rather than leaving the reader to infer it from the checks.
func logSubmitted(ctx context.Context, ch entity.Change, actor, dryError string) {
	switch ch.Status {
	case constants.ChangeFailed:
		logging.WarnWithCtx(ctx, "change rejected: "+dryError, "connection", ch.Connection, "kind", ch.Kind, "actor", actor)
	case constants.ChangeUnverified:
		logging.WarnWithCtx(ctx, "change recorded unverified: the driver cannot dry-run it",
			"change", ch.ID, "connection", ch.Connection, "kind", ch.Kind, "actor", actor)
	default:
		logging.InfoWithCtx(ctx, "change validated",
			"change", ch.ID, "connection", ch.Connection, "kind", ch.Kind, "scratch", ch.ScratchSchema, "actor", actor)
	}
}

// SwaggerMongoChangeBody emits the MongoDB submit body schema.
//
// @Param body body entity.MongoChangeRequest true "mongodb request"
func SwaggerMongoChangeBody() {}
