// Command seed migrates the database and loads the embedded master data.
// With -sec-file it also imports NYSE/Nasdaq issuers from a downloaded
// https://www.sec.gov/files/company_tickers_exchange.json. Safe to run
// repeatedly.
package main

import (
	"context"
	"flag"
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
	secFile := flag.String("sec-file", "", "path to SEC company_tickers_exchange.json to import")
	flag.Parse()

	cfg := config.Load()
	if err := store.MigrateUp(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		return err
	}
	if *secFile == "" {
		return nil
	}
	b, err := os.ReadFile(*secFile)
	if err != nil {
		return err
	}
	companies, err := catalog.ParseSECTickers(b)
	if err != nil {
		return err
	}
	return catalog.ImportSEC(ctx, pool, companies)
}
