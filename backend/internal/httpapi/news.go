package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// Public data changes at most every few minutes; let browsers and the
// Next.js server cache briefly.
const publicCache = "public, max-age=60"

type handlers struct {
	svc *service.Service
}

func (h handlers) news(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	nq := service.NewsQuery{
		Ticker: q.Get("ticker"), Index: q.Get("index"), Source: q.Get("source"), Cursor: q.Get("cursor"),
	}
	if k := q.Get("kind"); k != "" {
		nq.Kinds = strings.Split(k, ",")
	}
	var err error
	if nq.From, err = optTime(q.Get("from")); err != nil {
		writeError(w, http.StatusBadRequest, "from must be RFC 3339")
		return
	}
	if nq.To, err = optTime(q.Get("to")); err != nil {
		writeError(w, http.StatusBadRequest, "to must be RFC 3339")
		return
	}
	if s := q.Get("limit"); s != "" {
		if nq.Limit, err = strconv.Atoi(s); err != nil || nq.Limit == 0 {
			writeError(w, http.StatusBadRequest, "limit must be an integer between 1 and 100")
			return
		}
	}

	page, err := h.svc.ListNews(r.Context(), nq)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Cache-Control", publicCache)
	writeJSON(w, http.StatusOK, toNewsPage(page))
}

func (h handlers) searchInstruments(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.SearchInstruments(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := instrumentListDTO{Items: make([]instrumentDTO, len(items))}
	for i, it := range items {
		out.Items[i] = toInstrument(it)
	}
	w.Header().Set("Cache-Control", publicCache)
	writeJSON(w, http.StatusOK, out)
}

func (h handlers) instrument(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Instrument(r.Context(), r.PathValue("ticker"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Cache-Control", publicCache)
	writeJSON(w, http.StatusOK, instrumentDetailDTO{instrumentDTO: toInstrument(d.Instrument), Country: d.Country, Indices: d.Indices})
}

func (h handlers) sources(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.Sources(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := sourceListDTO{Items: make([]sourceDTO, len(items))}
	for i, s := range items {
		out.Items[i] = sourceDTO{Key: s.Key, Name: s.Name, Market: s.Market, Kind: s.Kind}
	}
	w.Header().Set("Cache-Control", publicCache)
	writeJSON(w, http.StatusOK, out)
}

func optTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	var verr *service.ValidationError
	switch {
	case errors.As(err, &verr):
		writeError(w, http.StatusBadRequest, verr.Msg)
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		slog.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error") // never leak details
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorDTO{Error: msg})
}
