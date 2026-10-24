package uat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
)

// erdBody is the wire shape of the ERD endpoint (only the asserted fields).
type erdBody struct {
	Connection struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
	} `json:"connection"`
	Tables []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Columns []struct {
			Name       string `json:"name"`
			PrimaryKey bool   `json:"primaryKey"`
			References *struct {
				Table string `json:"table"`
			} `json:"references"`
		} `json:"columns"`
	} `json:"tables"`
}

func (w *world) erdSteps(sc *godog.ScenarioContext) {
	sc.Step(`^the ERD of "([^"]*)" marks "([^"]*)" as a primary key$`, w.erdPrimaryKey)
	sc.Step(`^the ERD of "([^"]*)" links "([^"]*)" to "([^"]*)"$`, w.erdLinks)
}

// erd loads one connection's schema graph.
func (w *world) erd(ctx context.Context, connection string) (erdBody, string, error) {
	var body erdBody
	code, raw, err := w.raw("/external/v1/engine/sql/connections/" + connection + "/erd")
	if err != nil {
		return body, "", err
	}
	if code != http.StatusOK {
		return body, raw, fmt.Errorf("erd: %d %s", code, raw)
	}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return body, raw, fmt.Errorf("erd decode: %w", err)
	}
	if body.Connection.Name != connection {
		return body, raw, fmt.Errorf("erd names connection %q, want %q", body.Connection.Name, connection)
	}
	if strings.Contains(strings.ToLower(raw), `"password"`) {
		return body, raw, fmt.Errorf("the erd must not carry a password: %s", raw)
	}
	seen := make(map[string]bool, len(body.Tables))
	for _, t := range body.Tables {
		if t.ID == "" {
			return body, raw, fmt.Errorf("erd table %q has no graph id", t.Name)
		}
		if seen[t.ID] {
			return body, raw, fmt.Errorf("erd graph id %q is not unique", t.ID)
		}
		seen[t.ID] = true
	}
	return body, raw, nil
}

// erdPrimaryKey asserts the catalog's primary key reaches the diagram.
func (w *world) erdPrimaryKey(ctx context.Context, connection, column string) error {
	body, _, err := w.erd(ctx, connection)
	if err != nil {
		return err
	}
	for _, t := range body.Tables {
		for _, c := range t.Columns {
			if c.Name == column && c.PrimaryKey {
				return nil
			}
		}
	}
	return fmt.Errorf("no column %q is marked as a primary key in %+v", column, body.Tables)
}

// erdLinks asserts a declared foreign key reaches the graph as a reference.
func (w *world) erdLinks(ctx context.Context, connection, child, parent string) error {
	body, _, err := w.erd(ctx, connection)
	if err != nil {
		return err
	}
	for _, t := range body.Tables {
		if t.Name != child {
			continue
		}
		for _, c := range t.Columns {
			if c.References != nil && c.References.Table == parent {
				return nil
			}
		}
	}
	return fmt.Errorf("no reference from %q to %q in %+v", child, parent, body.Tables)
}
