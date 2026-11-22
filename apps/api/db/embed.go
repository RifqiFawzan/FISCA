// Package db embeds the SQL migrations and development seed so that the API
// cannot start without the schema it was built against.
package db

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed seed/*.sql
var seedFS embed.FS

// Migrations returns the goose migrations directory.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic("db: migrations dir missing: " + err.Error())
	}
	return sub
}

// Seed returns the development seed directory.
func Seed() fs.FS {
	sub, err := fs.Sub(seedFS, "seed")
	if err != nil {
		panic("db: seed dir missing: " + err.Error())
	}
	return sub
}
