// Package api holds the OpenAPI description of this service and embeds it so
// the binary can serve its own documentation.
package api

import _ "embed"

// Spec is the contents of openapi.yaml, read at build time.
//
//go:embed openapi.yaml
var Spec []byte
