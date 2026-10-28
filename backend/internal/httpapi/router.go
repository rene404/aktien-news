// Package httpapi holds the public REST handlers. Handlers stay thin: they
// parse input, call a service and shape the response.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// NewRouter wires all public routes.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("encode response", "err", err)
	}
}
