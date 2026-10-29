// Package config reads runtime settings from the environment. Every setting
// has a local default so the stack boots without any API keys.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// DefaultTextRetention is how long stored article text is kept.
const DefaultTextRetention = 30 * 24 * time.Hour

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	UserAgent   string
	// SECUserAgent identifies us to sec.gov, which requires a contact
	// address ("aktien-news admin@example.com"). Empty: SEC sources are skipped.
	SECUserAgent string
	// TextRetention: stored article text is purged this long after it was
	// fetched. TEXT_RETENTION takes a Go duration ("720h") or days ("30d").
	TextRetention time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:      getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/aktien_news?sslmode=disable"),
		UserAgent:     getenv("USER_AGENT", "aktien-news-bot/0.1 (+https://aktien-news.at)"),
		SECUserAgent:  os.Getenv("SEC_USER_AGENT"),
		TextRetention: retention(os.Getenv("TEXT_RETENTION")),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func retention(v string) time.Duration {
	if v == "" {
		return DefaultTextRetention
	}
	d, err := time.ParseDuration(v)
	if days, ok := strings.CutSuffix(v, "d"); ok {
		var n int
		n, err = strconv.Atoi(days)
		d = time.Duration(n) * 24 * time.Hour
	}
	if err != nil || d <= 0 {
		slog.Warn("invalid TEXT_RETENTION, using default", "value", v, "default", DefaultTextRetention)
		return DefaultTextRetention
	}
	return d
}
