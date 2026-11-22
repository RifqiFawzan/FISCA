// Command migrate applies goose migrations embedded in the API binary.
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
//	go run ./cmd/migrate status
//	go run ./cmd/migrate reset
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	apidb "github.com/fisca-app/fisca/apps/api/db"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/envfile"
)

func main() {
	envfile.Load(".env", "../../.env")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(apidb.Migrations())
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose dialect: %v", err)
	}

	if err := run(db, command); err != nil {
		log.Fatalf("migrate %s: %v", command, err)
	}
}

func run(db *sql.DB, command string) error {
	switch command {
	case "up":
		return goose.Up(db, ".")
	case "up-by-one":
		return goose.UpByOne(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	case "version":
		version, err := goose.GetDBVersion(db)
		if err != nil {
			return err
		}
		fmt.Printf("current version: %d\n", version)
		return nil
	case "reset":
		return goose.Reset(db, ".")
	default:
		return fmt.Errorf("unknown command %q (use up|up-by-one|down|status|version|reset)", command)
	}
}
