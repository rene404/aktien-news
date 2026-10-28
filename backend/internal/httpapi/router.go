// Package httpapi holds the public REST handlers. Handlers stay thin: they
// parse input, call a service and shape the response.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// NewRouter wires all public routes. svc may be nil in tests that only
// exercise /healthz.
func NewRouter(svc *service.Service) http.Handler {
	h := handlers{svc: svc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /api/v1/news", h.news)
	mux.HandleFunc("GET /api/v1/instruments", h.searchInstruments)
	mux.HandleFunc("GET /api/v1/instruments/{ticker}", h.instrument)
	mux.HandleFunc("GET /api/v1/sources", h.sources)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("encode response", "err", err)
	}
}
