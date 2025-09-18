// +build embed

package main

import "embed"

// Embed frontend static files (only available during Docker build)
//go:embed all:frontend/dist
var frontendFS embed.FS