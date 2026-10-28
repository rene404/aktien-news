// Package store owns all database access: goose migrations, sqlc queries
// (package db) and the generated models.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateUp applies all pending migrations.
func MigrateUp(ctx context.Context, dsn string) error {
	return withProvider(ctx, dsn, func(p *goose.Provider) error {
		_, err := p.Up(ctx)
		return err
	})
}

// MigrateDown rolls back every migration.
func MigrateDown(ctx context.Context, dsn string) error {
	return withProvider(ctx, dsn, func(p *goose.Provider) error {
		_, err := p.DownTo(ctx, 0)
		return err
	})
}

func withProvider(ctx context.Context, dsn string, fn func(*goose.Provider) error) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()
	if err := conn.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, conn, sub)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}
	return fn(p)
}
