package migrations

import "embed"

// Files contains all versioned SQL migrations bundled into the binaries.
//
//go:embed *.sql
var Files embed.FS
