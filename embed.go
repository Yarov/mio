// Package mio provides embedded assets for the Mio binary.
// These assets are compiled into the binary so they're available
// regardless of where the binary is installed (e.g., /usr/local/bin/mio).
package mio

import "embed"

// Assets contains all embeddable files: protocols, hooks, output-styles,
// statusline, skills, and plugins.
//
//go:embed protocols hooks output-styles statusline.sh skills plugins
var Assets embed.FS
