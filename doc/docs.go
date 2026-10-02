// Package doc embeds the master's OpenAPI spec; the master serves it at
// /openapi.yaml so frontends and tooling fetch one source of truth.
package doc

import (
	_ "embed"
)

//go:embed swagger.yaml
var SwaggerYAML []byte