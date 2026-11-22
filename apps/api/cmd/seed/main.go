// Command seed loads development fixtures. It is idempotent and refuses to run
// when APP_ENV=production.
//
//	go run ./cmd/seed
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	apidb "github.com/fisca-app/fisca/apps/api/db"
	"github.com/fisca-app/fisca/apps/api/internal/infrastructure/envfile"
)

func main() {
	envfile.Load(".env", "../../.env")

	if os.Getenv("APP_ENV") == "production" {
		log.Fatal("refusing to seed development data when APP_ENV=production")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	entries, err := fs.ReadDir(apidb.Seed(), ".")
	if err != nil {
		log.Fatalf("read seed dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		raw, err := fs.ReadFile(apidb.Seed(), entry.Name())
		if err != nil {
			log.Fatalf("read %s: %v", entry.Name(), err)
		}

		for _, statement := range splitStatements(string(raw)) {
			if _, err := pool.Exec(ctx, statement); err != nil {
				log.Fatalf("seed %s: %v", entry.Name(), err)
			}
		}
		fmt.Printf("seeded %s\n", entry.Name())
	}

	fmt.Println("seed complete")
}

// splitStatements splits a seed file on semicolons. Seed files must not contain
// dollar-quoted blocks (they do not need any).
func splitStatements(raw string) []string {
	chunks := strings.Split(raw, ";")
	statements := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if !hasSQL(chunk) {
			continue
		}
		statements = append(statements, strings.TrimSpace(chunk))
	}
	return statements
}

func hasSQL(chunk string) bool {
	for _, line := range strings.Split(chunk, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			return true
		}
	}
	return false
}
