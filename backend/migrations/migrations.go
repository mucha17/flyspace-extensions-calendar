// Package migrations embeds the SQL migration files applied at startup.
package migrations

import "embed"

// FS holds the migration files. Up files are named "<version>.up.sql" and applied in lexical order.
//
//go:embed *.sql
var FS embed.FS
