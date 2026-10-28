// Package testdb gives every test its own throwaway Postgres database on the
// server named by TEST_DATABASE_URL, so tests never share state.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/store"
)

// NewEmpty creates an empty database and returns its DSN. It is dropped when
// the test ends.
func NewEmpty(t *testing.T) string {
	t.Helper()
	adminDSN := os.Getenv("TEST_DATABASE_URL")
	if adminDSN == "" {
		t.Fatal("TEST_DATABASE_URL is not set; run tests via `docker compose --profile test run --rm backend-test`")
	}
	ctx := context.Background()

	admin, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		t.Fatalf("connect admin db: %v", err)
	}
	defer admin.Close(ctx)

	name := "t_" + randomHex(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), adminDSN)
		if err != nil {
			t.Logf("cleanup connect: %v", err)
			return
		}
		defer c.Close(context.Background())
		if _, err := c.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Logf("drop database %s: %v", name, err)
		}
	})

	u, err := url.Parse(adminDSN)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}

// New creates a migrated database and returns a pool to it.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := NewEmpty(t)
	ctx := context.Background()
	if err := store.MigrateUp(ctx, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func randomHex(t *testing.T) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}
