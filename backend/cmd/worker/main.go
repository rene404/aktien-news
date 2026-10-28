// Command worker runs ingestion: every tick it processes the sources whose
// fetch interval has elapsed. Intervals live in the database, so restarts
// never cause early refetches.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/config"
	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/fetch"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
	"github.com/rene404/aktien-news/backend/internal/store"
)

const tick = time.Minute

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("worker failed", "err", err)
		os.Exit(1)
	}
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

	runner := &ingest.Runner{
		Pool:    pool,
		Fetcher: fetch.New(cfg.UserAgent, 30*time.Second),
		Parsers: map[string]ingest.Parser{
			"finanznachrichten": finanznachrichten.Parse,
		},
		Now: time.Now,
	}

	slog.Info("worker started", "tick", tick)
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		if err := runner.RunDue(ctx); err != nil && ctx.Err() == nil {
			slog.Error("ingest tick failed", "err", err)
		}
		select {
		case <-ctx.Done():
			slog.Info("worker stopping")
			return nil
		case <-t.C:
		}
	}
}
