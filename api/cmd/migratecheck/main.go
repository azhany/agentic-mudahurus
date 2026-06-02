// Command migratecheck applies all up migrations against DATABASE_URL and exits.
// Used as the CI "migration dry-run check gate" (MH-003) and for ops.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mudahurus/api/internal/config"
	"github.com/mudahurus/api/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
	fmt.Println("migrations OK")
}
