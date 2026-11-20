package handlers

import (
	"net/http"
	"strconv"
	"time"

	"noxis/cmd/http/middlewares"
	"noxis/cmd/http/respond"
	"noxis/internal/app/proposal"
	"noxis/internal/domain"
)

// proposalWire is a proposal on the wire. It carries the statement and its rollback because a
// reviewer has to read both before deciding.
type proposalWire struct {
	Ref          string     `json:"ref"`
	TargetRef    string     `json:"target_ref"`
	Title        string     `json:"title"`
	Kind         string     `json:"kind"`
	Query        string     `json:"query"`
	Rollback     string     `json:"rollback"`
	Note         string     `json:"note"`
	Status       string     `json:"status"`
	Tier         string     `json:"tier"`
	Irreversible bool       `json:"irreversible"`
	ProposedBy   string     `json:"proposed_by"`
	RehearsalRef string     `json:"rehearsal_ref,omitempty"`
	DecidedBy    string     `json:"decided_by,omitempty"`
	DecidedAt    *time.Time `json:"decided_at,omitempty"`
	DecidedNote  string     `json:"decided_note,omitempty"`
	AppliedBy    string     `json:"applied_by,omitempty"`
	AppliedAt    *time.Time `json:"applied_at,omitempty"`
	Failure      string     `json:"failure,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// rehearsalWire is the evidence behind a proposal: diffs, the note that says how far the claim
// goes, and the tier it was actually run at.
type rehearsalWire struct {
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

func toProposalWire(p domain.Proposal) proposalWire {
	return proposalWire{
		Ref: p.Ref, TargetRef: p.TargetRef, Title: p.Title, Kind: string(p.Kind), Query: p.Query,
		Rollback: p.Rollback, Note: p.Note, Status: string(p.Status), Tier: string(p.Tier),
		Irreversible: p.Irreversible, ProposedBy: p.ProposedBy, RehearsalRef: p.RehearsalRef,
		DecidedBy: p.DecidedBy, DecidedAt: p.DecidedAt, DecidedNote: p.DecidedNote,
		AppliedBy: p.AppliedBy, AppliedAt: p.AppliedAt, Failure: p.Failure,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toRehearsalWire(r domain.Rehearsal) rehearsalWire {
	return rehearsalWire{
		Ref: r.Ref, ProposalRef: r.ProposalRef, Tier: string(r.Tier), Status: string(r.Status),
		SchemaDiff: r.SchemaDiff, RowDiff: r.RowDiff, Plan: r.Plan, Irreversible: r.Irreversible,
		Note: r.Note, DurationMS: r.DurationMS, Error: r.Error, CreatedAt: r.CreatedAt,
	}
}

// CreateProposal takes a statement and its rollback, stores the proposal and rehearses it. A
// rehearsal that does not prove the change still answers 201 with the proposal: the artifact is the
// point, and `failed` is a verdict, not a transport error.
func (h *Handlers) CreateProposal(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title    string `json:"title"`
		Query    string `json:"query"`
		Rollback string `json:"rollback"`
		Note     string `json:"note"`
		Tier     string `json:"tier"`
	}
	if err := decode(w, r, &body); err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	out, err := h.Proposals.Create.Run(r.Context(), proposal.CreateInput{
		TargetRef: r.PathValue("ref"),
		Title:     body.Title,
		Query:     body.Query,
		Rollback:  body.Rollback,
		Note:      body.Note,
		Tier:      body.Tier,
		Actor:     middlewares.Caller(r.Context()).Name,
	})
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	respond.Created(w, struct {
		proposalWire
		Rehearsal *rehearsalWire `json:"rehearsal,omitempty"`
	}{proposalWire: toProposalWire(out.Proposal), Rehearsal: wireOf(out.Rehearsal)})
}

func wireOf(rehearsal domain.Rehearsal) *rehearsalWire {
	if rehearsal.Ref == "" {
		return nil
	}
	wire := toRehearsalWire(rehearsal)
	return &wire
}

// ListProposals answers one page of the queue. `mine` is the caller's own name, not a parameter.
func (h *Handlers) ListProposals(w http.ResponseWriter, r *http.Request) {
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
	wires := make([]proposalWire, 0, len(page.Proposals))
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
func (h *Handlers) GetProposal(w http.ResponseWriter, r *http.Request) {
	out, err := h.Proposals.Get.Run(r.Context(), r.PathValue("ref"))
	if err != nil {
		respond.Fail(w, h.Log, err)
		return
	}
	body := struct {
		proposalWire
		Rehearsal *rehearsalWire `json:"rehearsal,omitempty"`
	}{proposalWire: toProposalWire(out.Proposal)}
	if out.HasRehearsal {
		body.Rehearsal = wireOf(out.Rehearsal)
	}
	respond.OK(w, body)
}

// RehearseProposal re-runs the proof for a proposal that has not been decided.
func (h *Handlers) RehearseProposal(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tier string `json:"tier,omitempty"`
	}
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
	respond.OK(w, struct {
		proposalWire
		Rehearsal *rehearsalWire `json:"rehearsal,omitempty"`
	}{proposalWire: toProposalWire(updated), Rehearsal: wireOf(rehearsal)})
}
