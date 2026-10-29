// Command api serves the public REST API and, with MCP_AUTH_TOKEN set, the
// internal MCP endpoint for mcp-hub on a separate port.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/config"
	"github.com/rene404/aktien-news/backend/internal/httpapi"
	"github.com/rene404/aktien-news/backend/internal/mcpserver"
	"github.com/rene404/aktien-news/backend/internal/service"
)

func main() {
	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("database config", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	svc := service.New(pool)
	servers := []*http.Server{{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}}
	// MCP returns sentiment, so it runs on its own port (never published) and
	// only with a token; without one the public API still boots.
	if cfg.MCPAuthToken != "" {
		mux := http.NewServeMux()
		mux.Handle("/mcp", mcpserver.Handler(svc, cfg.MCPAuthToken))
		servers = append(servers, &http.Server{Addr: cfg.MCPAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second})
	} else {
		slog.Info("MCP_AUTH_TOKEN not set, MCP server disabled")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	for _, srv := range servers {
		go func() {
			slog.Info("listening", "addr", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("server failed", "addr", srv.Addr, "err", err)
				os.Exit(1)
			}
		}()
	}

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, srv := range servers {
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "addr", srv.Addr, "err", err)
		}
	}
}
