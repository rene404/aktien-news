package config_test

import (
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/config"
)

func TestTextRetention(t *testing.T) {
	cases := map[string]time.Duration{
		"":      30 * 24 * time.Hour, // default
		"168h":  7 * 24 * time.Hour,
		"14d":   14 * 24 * time.Hour,
		"bogus": 30 * 24 * time.Hour, // unparseable falls back
		"-1h":   30 * 24 * time.Hour, // must be positive
	}
	for in, want := range cases {
		t.Setenv("TEXT_RETENTION", in)
		if got := config.Load().TextRetention; got != want {
			t.Errorf("TEXT_RETENTION=%q: %v, want %v", in, got, want)
		}
	}
}

func TestMCP(t *testing.T) {
	t.Setenv("MCP_ADDR", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	if c := config.Load(); c.MCPAddr != ":8081" || c.MCPAuthToken != "" {
		t.Fatalf("defaults: addr %q token %q, want :8081 and none (MCP off)", c.MCPAddr, c.MCPAuthToken)
	}
	t.Setenv("MCP_ADDR", ":9000")
	t.Setenv("MCP_AUTH_TOKEN", "tok")
	if c := config.Load(); c.MCPAddr != ":9000" || c.MCPAuthToken != "tok" {
		t.Fatalf("from env: addr %q token %q", c.MCPAddr, c.MCPAuthToken)
	}
}
