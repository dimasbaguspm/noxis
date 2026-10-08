package controller

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"noxis/internal/common/idgen"
	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/interfaces"
	"noxis/internal/services/sql"
)

// The batch flow: several `.sql` files uploaded together become several
// changes sharing one batch id. "One file = one statement" is enforced here,
// so a file that is really a script is refused before it can be recorded.

// batchFile is one uploaded .sql file: its name and the statements it carries.
type batchFile struct {
	Name     string `json:"name"`
	Query    string `json:"query"`
	Rollback string `json:"rollback"`
}

type batchRequest struct {
	Connection string      `json:"connection"`
	Notes      string      `json:"notes"`
	Files      []batchFile `json:"files"`
}

// batchFileResult is one file's outcome: the recorded change when it verified,
// or why it did not.
type batchFileResult struct {
	Name   string        `json:"name"`
	OK     bool          `json:"ok"`
	Error  string        `json:"error,omitempty"`
	Change entity.Change `json:"change,omitempty"`
}

type batchResult struct {
	BatchID string            `json:"batchId"`
	Results []batchFileResult `json:"results"`
}

// submitBatch raises one change per file, all sharing a batch id, and
// verifies each through the normal change path.
func (d Deps) submitBatch(ctx context.Context, sq interfaces.SQLModule, body batchRequest) (out batchResult, err error) {
	ctx, end := d.span(ctx)
	defer func() { end(err) }()
	if len(body.Files) == 0 {
		return batchResult{}, fmt.Errorf("a batch needs at least one .sql file")
	}
	out.BatchID = idgen.UUID()
	out.Results = make([]batchFileResult, 0, len(body.Files))
	for _, f := range body.Files {
		out.Results = append(out.Results, d.submitBatchFile(ctx, sq, out.BatchID, body, f))
	}
	return out, nil
}

func (d Deps) submitBatchFile(ctx context.Context, sq interfaces.SQLModule, batchID string, body batchRequest, f batchFile) batchFileResult {
	res := batchFileResult{Name: f.Name}
	switch {
	case strings.TrimSpace(f.Query) == "":
		res.Error = "file is empty"
		return res
	case sql.StatementCount(f.Query) != 1:
		res.Error = "one file must hold exactly one statement"
		return res
	case strings.TrimSpace(f.Rollback) == "":
		res.Error = "rollback is required"
		return res
	}
	notes := strings.TrimSpace(body.Notes)
	if notes == "" {
		notes = "batch change"
	}
	if f.Name != "" {
		notes += " · " + f.Name
	}
	ch, err := d.submitChange(ctx, sq, entity.Change{
		Connection: body.Connection,
		Notes:      notes,
		Query:      f.Query,
		Rollback:   f.Rollback,
		BatchID:    batchID,
	})
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Change = ch
	if ch.Status == constants.ChangeFailed {
		res.Error = firstCheckError(ch.Checks)
		return res
	}
	res.OK = true
	return res
}

// firstCheckError names the first failing check, so the batch report says why
// a file did not verify.
func firstCheckError(checks []entity.ChangeCheck) string {
	for i := range checks {
		if !checks[i].OK {
			if checks[i].Error != "" {
				return checks[i].Error
			}
			return checks[i].Target + " failed"
		}
	}
	return "validation failed"
}

// @Summary Submit a batch of .sql files as changes sharing a batch id
// @Param engine path string true "engine family (sql)"
// @Param body body batchRequest true "connection, notes and one file per statement"
// @Success 200 {object} batchResult
// @Failure 400 {object} map[string]string
// @Router /v1/{engine}/changes/batch [post]
func SubmitChangeBatch(w http.ResponseWriter, r *http.Request) {
	handle(w, r, func(ctx context.Context, d Deps) (any, error) {
		sq, err := d.engineSQL(r)
		if err != nil {
			return nil, err
		}
		var body batchRequest
		if err := decode(r, &body); err != nil {
			return nil, err
		}
		return d.submitBatch(ctx, sq, body)
	})
}
