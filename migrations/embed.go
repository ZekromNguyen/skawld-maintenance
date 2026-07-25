package migrations

import "embed"

// FS contains all forward-only reviewed database migrations.
//
//go:embed *.sql
var FS embed.FS
