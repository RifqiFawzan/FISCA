// Package envfile loads the first .env file that exists. The repository keeps a
// single canonical .env at the root, so the API looks one level up as well.
package envfile

import (
	"os"

	"github.com/joho/godotenv"
)

// Load reads the given candidate paths in order and ignores missing files.
func Load(candidates ...string) {
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		_ = godotenv.Load(path)
		return
	}
}
