package web

import "embed"

// StaticFS contains the frontend static files (HTML, CSS, JS).
//
//go:embed static
var StaticFS embed.FS
