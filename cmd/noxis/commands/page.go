package commands

import (
	"fmt"
	"noxis/internal/client"
	"noxis/internal/constants"
)

// walkAll reads every page after the first one, through the cursor each answer carries. It is one
// function because paging is one rule: `--all` must not be a loop that every command writes for
// itself, and a list that stops early must stop where the API said it stops.
func walkAll[T any](first client.Page[T], next func(cursor string) (client.Page[T], error)) (client.Page[T], error) {
	page := first
	for pages := 1; page.NextCursor != ""; pages++ {
		if pages >= constants.MaxPages {
			return page, fmt.Errorf("stopped after %d pages: the server kept answering a cursor", constants.MaxPages)
		}
		more, err := next(page.NextCursor)
		if err != nil {
			return page, err
		}
		page.Items = append(page.Items, more.Items...)
		page.NextCursor = more.NextCursor
	}
	return page, nil
}
