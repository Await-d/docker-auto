// +build !embed

package main

import "embed"

// Empty filesystem for development builds
var frontendFS embed.FS