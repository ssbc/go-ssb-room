package sqlite

import migrate "github.com/rubenv/sql-migrate"

//go:generate go run -tags=dev migrations_generate.go

var migrationSource = &migrate.HttpFileSystemMigrationSource{
	FileSystem: Migrations,
}
