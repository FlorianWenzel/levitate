// Package migrations embeds the SQL migration files into the binary so the
// produced image is self-contained and doesn't need a host-mounted directory.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
