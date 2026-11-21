package http

import (
	"fmt"
	"net/http"
	"noxis/cmd/noxis/http/respond"
	"noxis/internal/constants"
	"path"
	"strings"
)

// apiOwned is the namespace this mux owns. A path under one of these is a route or a mistake, never a
// page: a client that asked for /v1/targts and got HTML would have to parse a document to learn its
// own typo, and a script that asked for a page and got JSON would have to guess.
var apiOwned = []string{"/v1", "/openapi.json", "/openapi.yaml", "/healthz"}

// console serves the browser from the API's own origin, and answers like the API wherever the API is
// the answer. It is what makes one process and one URL possible: the page and the calls it makes
// share an origin, so no browser is ever asked about CORS, and no second deployment has to know the
// address a person reaches the API on.
func console(files http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if owned(r.URL.Path) {
			respond.Refuse(w, http.StatusNotFound, constants.CodeNotFound,
				fmt.Sprintf("%s %s is not a route this API answers; the console is at /", r.Method, r.URL.Path))
			return
		}
		files.ServeHTTP(w, r)
	})
}

// owned reports whether the request fell into the API's namespace after the mux found no route for
// it. A path exactly at a prefix counts: `/v1` on its own is a mistake like any other.
func owned(p string) bool {
	clean := path.Clean(p)
	for _, prefix := range apiOwned {
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return true
		}
	}
	return false
}
