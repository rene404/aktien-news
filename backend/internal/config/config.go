// Package config reads runtime settings from the environment. Every setting
// has a local default so the stack boots without any API keys.
package config

import "os"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	UserAgent   string
}

func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/aktien_news?sslmode=disable"),
		UserAgent:   getenv("USER_AGENT", "aktien-news-bot/0.1 (+https://aktien-news.at)"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
