// Command seed migrates the database and loads the embedded master data.
// Safe to run repeatedly.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/config"
	"github.com/rene404/aktien-news/backend/internal/store"
	"github.com/rene404/aktien-news/backend/seed"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
	slog.Info("seed complete")
}

func run(ctx context.Context) error {
	cfg := config.Load()
	if err := store.MigrateUp(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return catalog.Seed(ctx, pool, seed.FS)
}
