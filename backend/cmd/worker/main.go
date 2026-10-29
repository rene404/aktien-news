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
	"github.com/rene404/aktien-news/backend/internal/ingest/pressrss"
	"github.com/rene404/aktien-news/backend/internal/ingest/sec"
	"github.com/rene404/aktien-news/backend/internal/store"
)

const (
	tick          = time.Minute
	purgeInterval = time.Hour // expired article_texts are deleted this often
)

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

	fetcher := fetch.New(cfg.UserAgent, 30*time.Second)
	runner := &ingest.Runner{
		Pool:    pool,
		Fetcher: fetcher,
		Parsers: map[string]ingest.Parser{
			"finanznachrichten": finanznachrichten.Parse,
			"pressrss":          pressrss.Parse,
			"sec":               sec.Parse,
		},
		TDM: fetcher,
		Now: time.Now,
	}
	if cfg.SECUserAgent == "" {
		runner.Skip = map[string]string{"sec": "SEC_USER_AGENT not set"}
	} else {
		runner.Fetchers = map[string]ingest.Fetcher{"sec": fetch.New(cfg.SECUserAgent, 30*time.Second)}
	}

	// Backfill problems must not keep ingestion from starting.
	if failed, err := runner.ClusterUnassigned(ctx); err != nil {
		slog.Error("cluster backfill failed", "err", err)
	} else if failed > 0 {
		slog.Warn("cluster backfill incomplete", "failed", failed)
	}

	slog.Info("worker started", "tick", tick)
	t := time.NewTicker(tick)
	defer t.Stop()
	var lastPurge time.Time
	for {
		if err := runner.RunDue(ctx); err != nil && ctx.Err() == nil {
			slog.Error("ingest tick failed", "err", err)
		}
		if now := time.Now(); now.Sub(lastPurge) >= purgeInterval {
			lastPurge = now
			if n, err := ingest.PurgeTexts(ctx, pool, now); err != nil && ctx.Err() == nil {
				slog.Error("text purge failed", "err", err)
			} else if n > 0 {
				slog.Info("purged expired texts", "rows", n)
			}
		}
		select {
		case <-ctx.Done():
			slog.Info("worker stopping")
			return nil
		case <-t.C:
		}
	}
}
