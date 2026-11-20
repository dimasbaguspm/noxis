package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"noxis/internal/app/target"
	"noxis/internal/domain"
	"noxis/internal/httpapi/middlewares"
	"noxis/internal/httpapi/respond"
)

// maxBody bounds a request body. A registration is a few hundred bytes; anything larger is
// a mistake or an attack.
const maxBody = 1 << 20

// targetWire is what a client sees. The sealed password is not a field here, so it cannot
// be leaked by a later refactor: `has_password` is the only trace of it.
type targetWire struct {
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

func toWire(t domain.Target) targetWire {
	options := t.Options
	if options == nil {
		options = map[string]string{}
	}
	return targetWire{
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

func toWires(targets []domain.Target) []targetWire {
	out := make([]targetWire, 0, len(targets))
	for _, t := range targets {
		out = append(out, toWire(t))
	}
	return out
}

// CreateTarget registers one target. It needs `admin`: a token that may read must not be
// able to widen what the server points at.
func (h *Handlers) CreateTarget(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string            `json:"name"`
		Driver   string            `json:"driver"`
		Host     string            `json:"host"`
		Port     int               `json:"port"`
		Database string            `json:"database"`
		Username string            `json:"username"`
		Password string            `json:"password"`
		TLSMode  string            `json:"tls_mode"`
		Options  map[string]string `json:"options"`
	}
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	created, err := h.Targets.Create.Run(r.Context(), target.CreateInput{
		Name:     body.Name,
		Driver:   body.Driver,
		Host:     body.Host,
		Port:     body.Port,
		Database: body.Database,
		Username: body.Username,
		Password: body.Password,
		TLSMode:  body.TLSMode,
		Options:  body.Options,
		Actor:    middlewares.Caller(r.Context()).Name,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.Created(w, toWire(created))
}

// ListTargets answers one page of targets, newest first.
func (h *Handlers) ListTargets(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, err := intParam(query.Get("limit"))
	if err != nil {
		respond.Fail(w, h.Log, fmt.Errorf("%w: limit: %s", domain.ErrInvalid, err))
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
func (h *Handlers) GetTarget(w http.ResponseWriter, r *http.Request) {
	found, err := h.Targets.Get.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, toWire(found))
}

// UpdateTarget rotates a password, renames a target, or retunes its TLS mode and options.
func (h *Handlers) UpdateTarget(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     *string            `json:"name"`
		Username *string            `json:"username"`
		Password *string            `json:"password"`
		TLSMode  *string            `json:"tls_mode"`
		Options  *map[string]string `json:"options"`
	}
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
func (h *Handlers) ArchiveTarget(w http.ResponseWriter, r *http.Request) {
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
		return fmt.Errorf("%w: %s", domain.ErrInvalid, err)
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
func (h *Handlers) ProbeTarget(w http.ResponseWriter, r *http.Request) {
	result, err := h.Targets.Probe.Run(r.Context(), target.ProbeInput{
		Ref:   r.PathValue("ref"),
		Actor: middlewares.Caller(r.Context()).Name,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, struct {
		Target       targetWire          `json:"target"`
		Capabilities domain.Capabilities `json:"capabilities"`
	}{Target: toWire(result.Target), Capabilities: result.Capabilities})
}

// TargetCapabilities answers what this build may offer for the target's engine without opening
// a connection: the set is derived from the declared driver and, once probed, from the measured
// version.
func (h *Handlers) TargetCapabilities(w http.ResponseWriter, r *http.Request) {
	found, err := h.Targets.Get.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, domain.CapabilitiesFor(found.Driver, domain.MeasuredServer(found.Driver, found.ServerVersion)))
}
