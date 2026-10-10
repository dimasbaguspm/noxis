package logging

import "testing"

// The span-name convention is a contract: keep it stable.
func TestFormatSpanName(t *testing.T) {
	cases := map[string]string{
		"noxis/internal/repositories/sqlite.(*Store).List":    "Repository.Sqlite.Store.List",
		"noxis/internal/usecases.(*App).SubmitChange":        "Usecases.App.SubmitChange",
		"noxis/internal/modules/sql.(*DB).query":              "Modules.Sql.DB.query",
		"noxis/internal/common/logging.StartSpan":              "Common.Logging.StartSpan",
		"noxis/internal/entity.(*Connection).Redacted":         "Entity.Connection.Redacted",
		"noxis/cmd/noxis/server/api.(*Server).handle.func1":    "Noxis.Api.Server.handle.func1",
		"noxis/cmd/noxis.bootstrap":                            "Noxis.bootstrap",			
	}
	for in, want := range cases {
		if got := formatSpanName(in); got != want {
			t.Errorf("formatSpanName(%q) = %q, want %q", in, got, want)
		}
	}
}
