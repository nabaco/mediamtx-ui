// Package web embeds the compiled frontend assets.
// The build process copies frontend/dist → internal/web/dist before compiling.
package web

import "embed"

//go:embed all:dist
var FS embed.FS
