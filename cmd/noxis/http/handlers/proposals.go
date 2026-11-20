package handlers

import (
	"net/http"
	"noxis/cmd/noxis/http/middlewares"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/app/proposal"
	"noxis/internal/common/telemetry"
	"noxis/internal/constants"
	"noxis/internal/domain"
	"strconv"
	"time"
)

// ProposalDetail is a proposal and the rehearsal that last judged it: the evidence, together with
// the thing it is evidence about.
type ProposalDetail struct {
	Proposal
	Rehearsal *Rehearsal `json:"rehearsal,omitempty"`
}

// Proposal is a proposal on the wire. It carries the statement and its rollback because a
// reviewer has to read both before deciding.
type Proposal struct {
	Ref           string     `json:"ref"`
	TargetRef     string     `json:"target_ref"`
	Title         string     `json:"title"`
	Kind          string     `json:"kind"`
	Query         string     `json:"query"`
	Rollback      string     `json:"rollback"`
	Note          string     `json:"note"`
	Status        string     `json:"status"`
	Tier          string     `json:"tier"`
	Irreversible  bool       `json:"irreversible"`
	ProposedBy    string     `json:"proposed_by"`
	RehearsalRef  string     `json:"rehearsal_ref,omitempty"`
	DecidedBy     string     `json:"decided_by,omitempty"`
	DecidedAt     *time.Time `json:"decided_at,omitempty"`
	DecidedNote   string     `json:"decided_note,omitempty"`
	AppliedBy     string     `json:"applied_by,omitempty"`
	AppliedAt     *time.Time `json:"applied_at,omitempty"`
	Failure       string     `json:"failure,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	Progress      *Progress  `json:"progress,omitempty"`
	Algorithm     string     `json:"algorithm,omitempty"`
	RowsEstimated int64      `json:"rows_estimated,omitempty"`
	RowsAffected  int64      `json:"rows_affected,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Progress is what the engine said about a statement that was still running when the caller
// asked. Percent is -1 when the engine does not report one.
type Progress struct {
	Stage   string `json:"stage,omitempty"`
	Percent int    `json:"percent"`
}

// Rehearsal is the evidence behind a proposal: diffs, the note that says how far the claim
// goes, and the tier it was actually run at.
type Rehearsal struct {
	Ref          string    `json:"ref"`
	ProposalRef  string    `json:"proposal_ref"`
	Tier         string    `json:"tier"`
	Status       string    `json:"status"`
	SchemaDiff   string    `json:"schema_diff,omitempty"`
	RowDiff      string    `json:"row_diff,omitempty"`
	Plan         string    `json:"plan,omitempty"`
	Irreversible bool      `json:"irreversible"`
	Note         string    `json:"note"`
	DurationMS   int64     `json:"duration_ms"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// progressOf reports progress only while a statement is running: a finished apply has nothing to
// say about stages, and an absent field is clearer than a zero.
func progressOf(progress domain.Progress) *Progress {
	if progress.Stage == "" && progress.Percent < 0 {
		return nil
	}
	return &Progress{Stage: progress.Stage, Percent: progress.Percent}
}

func toProposalWire(p domain.Proposal) Proposal {
	return Proposal{
		Ref: p.Ref, TargetRef: p.TargetRef, Title: p.Title, Kind: string(p.Kind), Query: p.Query,
		Rollback: p.Rollback, Note: p.Note, Status: string(p.Status), Tier: string(p.Tier),
		Irreversible: p.Irreversible, ProposedBy: p.ProposedBy, RehearsalRef: p.RehearsalRef,
		DecidedBy: p.DecidedBy, DecidedAt: p.DecidedAt, DecidedNote: p.DecidedNote,
		AppliedBy: p.AppliedBy, AppliedAt: p.AppliedAt, Failure: p.Failure,
		StartedAt: p.StartedAt, Algorithm: string(p.Algorithm), RowsEstimated: p.RowsEstimated,
		RowsAffected: p.RowsAffected, Progress: progressOf(p.Progress),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toRehearsalWire(r domain.Rehearsal) Rehearsal {
	return Rehearsal{
		Ref: r.Ref, ProposalRef: r.ProposalRef, Tier: string(r.Tier), Status: string(r.Status),
		SchemaDiff: r.SchemaDiff, RowDiff: r.RowDiff, Plan: r.Plan, Irreversible: r.Irreversible,
		Note: r.Note, DurationMS: r.DurationMS, Error: r.Error, CreatedAt: r.CreatedAt,
	}
}

// CreateProposal takes a statement and its rollback, stores the proposal and rehearses it. A
// rehearsal that does not prove the change still answers 201 with the proposal: the artifact is the
// point, and `failed` is a verdict, not a transport error.
// @Summary Propose a change
// @Description Requires the propose scope. The proposal is rehearsed immediately: the schema is rebuilt in a throwaway database, the change is run there and the rollback must restore it.
// @Tags proposals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "target reference"
// @Param body body handlers.CreateProposalRequest true "the change and its rollback"
// @Param Idempotency-Key header string false "a name for this create; a retry with the same one answers the same proposal"
// @Success 200 {object} respond.Envelope "the proposal a retried create already made"
// @Success 201 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Router /v1/targets/{ref}/proposals [post]
func (h *Handlers) CreateProposal(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.CreateProposal")
	defer end(nil)
	r = r.WithContext(ctx)

	var body CreateProposalRequest
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	key, err := idempotencyKey(r)
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	out, err := h.Proposals.Create.Run(r.Context(), proposal.CreateInput{
		TargetRef:      r.PathValue("ref"),
		Title:          body.Title,
		Query:          body.Query,
		Rollback:       body.Rollback,
		Note:           body.Note,
		Tier:           body.Tier,
		Actor:          middlewares.Caller(r.Context()).Name,
		IdempotencyKey: key,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	detail := ProposalDetail{Proposal: toProposalWire(out.Proposal), Rehearsal: wireOf(out.Rehearsal)}
	// A retry answers the proposal the first call made, with the evidence that call produced.
	if out.Replayed {
		respond.OK(w, detail)
		return
	}
	respond.Created(w, detail)
}

func wireOf(rehearsal domain.Rehearsal) *Rehearsal {
	if rehearsal.Ref == "" {
		return nil
	}
	wire := toRehearsalWire(rehearsal)
	return &wire
}

// ListProposals answers one page of the queue. `mine` is the caller's own name, not a parameter.
// @Summary The proposal queue
// @Description Newest first, keyset-paged.
// @Tags proposals
// @Produce json
// @Security BearerAuth
// @Param limit query int false "page size"
// @Param cursor query string false "the last ref of the previous page"
// @Param target query string false "filter by target reference"
// @Param status query string false "filter by status"
// @Param kind query string false "ddl or dml"
// @Param mine query string false "only the caller's proposals"
// @Success 200 {object} respond.Envelope
// @Failure 400 {object} respond.Envelope
// @Router /v1/proposals [get]
func (h *Handlers) ListProposals(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ListProposals")
	defer end(nil)
	r = r.WithContext(ctx)

	query := r.URL.Query()
	limit := 0
	if raw := query.Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			respond.Fail(w, h.Log, err)
			return
		}
		limit = value
	}
	page, err := h.Proposals.List.Run(r.Context(), proposal.ListInput{
		Cursor: query.Get("cursor"),
		Limit:  limit,
		Target: query.Get("target"),
		Status: query.Get("status"),
		Kind:   query.Get("kind"),
		Mine:   mine(query.Get("mine"), middlewares.Caller(r.Context()).Name),
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	wires := make([]Proposal, 0, len(page.Proposals))
	for _, item := range page.Proposals {
		wires = append(wires, toProposalWire(item))
	}
	respond.Page(w, wires, page.NextCursor)
}

// mine turns `?mine=true` into the caller's name, so a list can be scoped without trusting a
// caller-supplied identity.
func mine(value, caller string) string {
	if value == "true" {
		return caller
	}
	return ""
}

// GetProposal reads one proposal with the rehearsal that last judged it.
// @Summary One proposal and its evidence
// @Tags proposals
// @Produce json
// @Security BearerAuth
// @Param ref path string true "proposal reference"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/proposals/{ref} [get]
func (h *Handlers) GetProposal(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.GetProposal")
	defer end(nil)
	r = r.WithContext(ctx)

	out, err := h.Proposals.Get.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	body := ProposalDetail{Proposal: toProposalWire(out.Proposal)}
	if out.HasRehearsal {
		body.Rehearsal = wireOf(out.Rehearsal)
	}
	respond.OK(w, body)
}

// RehearseProposal re-runs the proof for a proposal that has not been decided.
// @Summary Re-run the proof
// @Description Only for a proposal nobody has decided yet. Every run is stored, including the ones that failed.
// @Tags proposals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "proposal reference"
// @Param body body handlers.RehearseRequest false "an optional tier"
// @Success 200 {object} respond.Envelope
// @Failure 409 {object} respond.Envelope
// @Router /v1/proposals/{ref}/rehearse [post]
func (h *Handlers) RehearseProposal(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.RehearseProposal")
	defer end(nil)
	r = r.WithContext(ctx)

	var body RehearseRequest
	if r.ContentLength > 0 {
		if err := decode(w, r, &body); err != nil {
			respond.Fail(w, h.Log, err)
			return
		}
	}
	updated, rehearsal, err := h.Proposals.Rehearse.Run(r.Context(), proposal.RehearseInput{
		Ref:  r.PathValue("ref"),
		Tier: body.Tier,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, ProposalDetail{Proposal: toProposalWire(updated), Rehearsal: wireOf(rehearsal)})
}

// ApproveProposal records a human's approval. The caller's own token decides whether it may: the
// route asks for `approve`, and the use case refuses a token that also proposed it unless it is an
// admin — in which case the exception is journalled.
// @Summary Approve a proposal
// @Description Requires the approve scope. A token that also proposed the change is refused unless it holds admin, and the exception is journalled as its own line.
// @Tags proposals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "proposal reference"
// @Param body body handlers.DecisionRequest false "an optional note"
// @Success 200 {object} respond.Envelope
// @Failure 403 {object} respond.Envelope
// @Router /v1/proposals/{ref}/approve [post]
func (h *Handlers) ApproveProposal(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ApproveProposal")
	defer end(nil)
	r = r.WithContext(ctx)

	h.decide(w, r, true)
}

// RejectProposal records a refusal. A rejection may be the proposer withdrawing their own idea.
// @Summary Reject a proposal
// @Description Requires the approve scope. A rejection may be the proposer withdrawing their own idea.
// @Tags proposals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "proposal reference"
// @Param body body handlers.DecisionRequest false "an optional note"
// @Success 200 {object} respond.Envelope
// @Failure 409 {object} respond.Envelope
// @Router /v1/proposals/{ref}/reject [post]
func (h *Handlers) RejectProposal(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.RejectProposal")
	defer end(nil)
	r = r.WithContext(ctx)

	h.decide(w, r, false)
}

func (h *Handlers) decide(w http.ResponseWriter, r *http.Request, approve bool) {
	var body DecisionRequest
	if r.ContentLength > 0 {
		if err := decode(w, r, &body); err != nil {
			respond.Fail(w, h.Log, err)
			return
		}
	}
	caller := middlewares.Caller(r.Context())
	decided, err := h.Proposals.Decide.Run(r.Context(), proposal.DecideInput{
		Ref:          r.PathValue("ref"),
		Approve:      approve,
		Note:         body.Note,
		Actor:        caller.Name,
		ActorIsAdmin: caller.Has(constants.ScopeAdmin),
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.OK(w, toProposalWire(decided))
}

// ApplyProposal runs the approved statement. `confirm_destructive` is required when the rehearsal
// said the change cannot be undone.
// @Summary Apply an approved proposal
// @Description Requires the apply scope. DML runs in one transaction; DDL runs as one statement, because a receipt for an impossible rollback is a lie. A change the rehearsal called irreversible needs confirm_destructive. A table past the background threshold answers 202 and the progress is read from the proposal.
// @Tags proposals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param ref path string true "proposal reference"
// @Param body body handlers.ApplyRequest false "acknowledge a destructive change"
// @Success 200 {object} respond.Envelope
// @Success 202 {object} respond.Envelope
// @Failure 409 {object} respond.Envelope
// @Router /v1/proposals/{ref}/apply [post]
func (h *Handlers) ApplyProposal(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ApplyProposal")
	defer end(nil)
	r = r.WithContext(ctx)

	var body ApplyRequest
	if r.ContentLength > 0 {
		if err := decode(w, r, &body); err != nil {
			respond.Fail(w, h.Log, err)
			return
		}
	}
	out, err := h.Proposals.Apply.Run(r.Context(), proposal.ApplyInput{
		Ref:                r.PathValue("ref"),
		Actor:              middlewares.Caller(r.Context()).Name,
		ConfirmDestructive: body.ConfirmDestructive,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	if out.Started {
		// The statement outlives the request: the answer is "poll the proposal", not a held
		// connection and a client timeout that says nothing about the change.
		respond.Accepted(w, toProposalWire(out.Proposal))
		return
	}
	respond.OK(w, toProposalWire(out.Proposal))
}

// JournalEntry is one line of the append-only history, on the wire.
type JournalEntry struct {
	Ref         string            `json:"ref"`
	SubjectRef  string            `json:"subject_ref"`
	SubjectKind string            `json:"subject_kind"`
	Kind        string            `json:"kind"`
	Actor       string            `json:"actor"`
	Detail      map[string]string `json:"detail"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ProposalEvents answers the history of one proposal, newest first.
// @Summary The journal of one proposal
// @Description The append-only history: proposed, rehearsed, approved, applied, refused.
// @Tags proposals
// @Produce json
// @Security BearerAuth
// @Param ref path string true "proposal reference"
// @Success 200 {object} respond.Envelope
// @Failure 404 {object} respond.Envelope
// @Router /v1/proposals/{ref}/events [get]
func (h *Handlers) ProposalEvents(w http.ResponseWriter, r *http.Request) {
	ctx, end := telemetry.Span(r.Context(), "handlers.ProposalEvents")
	defer end(nil)
	r = r.WithContext(ctx)

	entries, err := h.Proposals.Events.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	wires := make([]JournalEntry, 0, len(entries))
	for _, entry := range entries {
		wires = append(wires, JournalEntry{
			Ref: entry.Ref, SubjectRef: entry.SubjectRef, SubjectKind: entry.SubjectKind,
			Kind: entry.Kind, Actor: entry.Actor, Detail: entry.Detail, CreatedAt: entry.CreatedAt,
		})
	}
	respond.OK(w, wires)
}
