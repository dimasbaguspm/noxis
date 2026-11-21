package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/app/target"
	"noxis/internal/common/telemetry"
	"noxis/internal/domain"
	"strconv"
	"time"
)

// maxBody bounds a request body. A registration is a few hundred bytes; anything larger is
// a mistake or an attack.
const maxBody = 1 << 20

// Target is what a client sees. The sealed password is not a field here, so it cannot
// be leaked by a later refactor: `has_password` is the only trace of it.
type Target struct {
	Ref         string            `json:"ref"`
	Name        string            `json:"name"`
	Driver      string            `json:"driver"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Database    string            `json:"database"`
	Username    string            `json:"username"`
	TLSMode     string            `json:"tls_mode"`
	Options     map[string]string `json:"options"`
	HasPassword bool              `json:"has_password"`
	CreatedBy   string            `json:"created_by"`
	// What the last probe measured. Empty until a probe succeeds.
	ServerVersion   string     `json:"server_version,omitempty"`
	LastProbeAt     *time.Time `json:"last_probe_at,omitempty"`
	LastProbeStatus string     `json:"last_probe_status,omitempty"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ProbeResult is what a probe answers: the target as it now stands, and what the engine said.
type ProbeResult struct {
	Target       Target              `json:"target"`
	Capabilities domain.Capabilities `json:"capabilities"`
}

func toWire(t domain.Target) Target {
	options := t.Options
	if options == nil {
		options = map[string]string{}
	}
	return Target{
		Ref:             t.Ref,
		Name:            t.Name,
		Driver:          t.Driver,
		Host:            t.Host,
		Port:            t.Port,
		Database:        t.Database,
		Username:        t.Username,
		TLSMode:         t.TLSMode,
		Options:         options,
		HasPassword:     t.Secret != "",
		CreatedBy:       t.CreatedBy,
		ServerVersion:   t.ServerVersion,
		LastProbeAt:     t.LastProbeAt,
		LastProbeStatus: t.LastProbeStatus,
		ArchivedAt:      t.ArchivedAt,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

func toWires(targets []domain.Target) []Target {
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		out = append(out, toWire(t))
	}
	return out
}

// CreateTarget registers one target. It needs `admin`: a token that may read must not be
// able to widen what the server points at.
// @Summary Register a target
// @Description Requires the admin scope. The password is sealed with SECRET_KEY and never returned.
// @Tags targets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body handlers.CreateTargetRequest true "the server to govern"
// @Param Idempotency-Key header string false "a name for this create; a retry with the same one answers the same row"
// @Success 200 {object} respond.Envelope "the row a retried create already made"
// @Success 201 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Router /v1/targets [post]
func (h *Handlers) CreateTarget(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.CreateTarget")
	defer end(nil)
	r = r.WithContext(ctx)

	var body CreateTargetRequest
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	key, err := idempotencyKey(r)
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	out, err := h.Targets.Create.Run(r.Context(), target.CreateInput{
		Name:           body.Name,
		Driver:         body.Driver,
		Host:           body.Host,
		Port:           body.Port,
		Database:       body.Database,
		Username:       body.Username,
		Password:       body.Password,
		TLSMode:        body.TLSMode,
		Options:        body.Options,
		Actor:          middlewares.Caller(r.Context()).Name,
		IdempotencyKey: key,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	// A retry answers the row the first call made, and says so with 200: 201 would claim a row this
	// request never created.
	if out.Replayed {
		respond.OK(w, toWire(out.Target))
		return
	}
	respond.Created(w, toWire(out.Target))
}

// TestTarget dials a configuration that is not stored yet, so the console can refuse a typo
// before it becomes a row. Nothing is written and nothing is journaled: a test that leaves a
// trace is a probe, and a probe is what a stored target gets.
// @Summary Test a configuration
// @Description Opens one connection with the credentials as typed and answers what the engine said. Requires the read scope.
// @Tags targets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body handlers.TestTargetRequest true "the configuration to dial"
// @Success 200 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Failure 422 {object} respond.Envelope
// @Router /v1/targets/test [post]
func (h *Handlers) TestTarget(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.TestTarget")
	defer end(nil)
	r = r.WithContext(ctx)

	var body TestTargetRequest
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	caps, err := h.Targets.Test.Run(r.Context(), target.TestInput{
		Driver:   body.Driver,
		Host:     body.Host,
		Port:     body.Port,
		Database: body.Database,
		Username: body.Username,
		Password: body.Password,
		TLSMode:  body.TLSMode,
		Options:  body.Options,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, caps)
}

// ListTargets answers one page of targets, newest first.
// @Summary List targets
// @Description One page of governed servers, newest first. Paging is a keyset cursor, never an offset.
// @Tags targets
// @Produce json
// @Security BearerAuth
// @Param limit query int false "page size"
// @Param cursor query string false "the last ref of the previous page"
// @Param driver query string false "mysql or mariadb"
// @Param q query string false "name filter"
// @Param include_archived query bool false "include archived targets"
// @Success 200 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Router /v1/targets [get]
func (h *Handlers) ListTargets(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ListTargets")
	defer end(nil)
	r = r.WithContext(ctx)

	query := r.URL.Query()
	limit, err := intParam(query.Get("limit"))
	if err != nil {
		respond.Fail(w, h.Log, fmt.Errorf("%w: limit: %w", domain.ErrInvalid, err))
		return
	}
	page, err := h.Targets.List.Run(r.Context(), target.ListInput{
		Cursor:          query.Get("cursor"),
		Limit:           limit,
		Driver:          query.Get("driver"),
		Query:           query.Get("q"),
		IncludeArchived: query.Get("include_archived") == "true",
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.Page(w, toWires(page.Targets), page.NextCursor)
}

// GetTarget reads one target by reference.
// @Summary Read one target
// @Tags targets
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/targets/{ref} [get]
func (h *Handlers) GetTarget(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.GetTarget")
	defer end(nil)
	r = r.WithContext(ctx)

	found, err := h.Targets.Get.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, toWire(found))
}

// UpdateTarget rotates a password, renames a target, or retunes its TLS mode and options.
// @Summary Patch a target
// @Description A field that is absent is a field nobody asked to change. Requires the admin scope.
// @Tags targets
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Param body body handlers.UpdateTargetRequest true "the fields to change"
// @Success 200 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Router /v1/targets/{ref} [patch]
func (h *Handlers) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.UpdateTarget")
	defer end(nil)
	r = r.WithContext(ctx)

	var body UpdateTargetRequest
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	updated, err := h.Targets.Update.Run(r.Context(), target.UpdateInput{
		Ref: r.PathValue("ref"),
		Patch: target.Patch{
			Name:     body.Name,
			Username: body.Username,
			Password: body.Password,
			TLSMode:  body.TLSMode,
			Options:  body.Options,
		},
		Actor: middlewares.Caller(r.Context()).Name,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, toWire(updated))
}

// ArchiveTarget puts a target away. Nothing is deleted.
// @Summary Archive a target
// @Description Nothing is deleted: the target is put away and the journal keeps the row.
// @Tags targets
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/targets/{ref} [delete]
func (h *Handlers) ArchiveTarget(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ArchiveTarget")
	defer end(nil)
	r = r.WithContext(ctx)

	archived, err := h.Targets.Archive.Run(r.Context(), target.ArchiveInput{
		Ref:   r.PathValue("ref"),
		Actor: middlewares.Caller(r.Context()).Name,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, toWire(archived))
}

// decode reads a JSON body into out, refusing what a client should not have sent: an
// unknown field (a typo that would otherwise be ignored) or anything over maxBody.
func decode(w http.ResponseWriter, r *http.Request, out any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: a json body is required", domain.ErrInvalid)
		}
		return fmt.Errorf("%w: %w", domain.ErrInvalid, err)
	}
	return nil
}

// intParam reads a query number, where absent is zero and negative is a mistake.
func intParam(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value < 0 {
		return 0, fmt.Errorf("must not be negative")
	}
	return value, nil
}

// ProbeTarget opens one connection, records what it measured on the target, and answers the
// capabilities it derived. A probe that cannot connect is a 422 whose message is the driver's
// own, and the target itself now says the attempt failed.
// @Summary Probe a target
// @Description Connects once, measures the engine and stores what it said.
// @Tags targets
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Success 200 {object} respond.Envelope
// @Failure 422 {object} respond.Envelope
// @Router /v1/targets/{ref}/probe [post]
func (h *Handlers) ProbeTarget(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ProbeTarget")
	defer end(nil)
	r = r.WithContext(ctx)

	result, err := h.Targets.Probe.Run(r.Context(), target.ProbeInput{
		Ref:   r.PathValue("ref"),
		Actor: middlewares.Caller(r.Context()).Name,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, ProbeResult{Target: toWire(result.Target), Capabilities: result.Capabilities})
}

// TargetCapabilities answers what this build may offer for the target's engine without opening
// a connection: the set is derived from the declared driver and, once probed, from the measured
// version.
// @Summary What the engine can do
// @Tags targets
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/targets/{ref}/capabilities [get]
func (h *Handlers) TargetCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.TargetCapabilities")
	defer end(nil)
	r = r.WithContext(ctx)

	found, err := h.Targets.Get.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, domain.CapabilitiesFor(found.Driver, domain.MeasuredServer(found.Driver, found.ServerVersion)))
}
