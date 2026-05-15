package migrations

import "embed"

// FS contains SQL migrations executed on application startup.
//
//go:embed *.sql
var FS embed.FS
