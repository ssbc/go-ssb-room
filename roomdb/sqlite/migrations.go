// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package sqlite

import (
	"embed"

	migrate "github.com/rubenv/sql-migrate"
)

// migrations is an embedded filesystem containing the sqlite migration files
//go:embed migrations/*
var migrations embed.FS

// needs https://github.com/rubenv/sql-migrate/pull/189 merged, using my branch until then
var migrationSource = &migrate.EmbedFileSystemMigrationSource{
	FileSystem: migrations,
	Root:       "migrations",
}
