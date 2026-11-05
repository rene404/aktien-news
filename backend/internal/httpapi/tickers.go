package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// GET /api/v1/tickers/top?from=&index=&kind=&limit= — the most-mentioned tickers.
func (h handlers) topTickers(w http.ResponseWriter, r *http.Request) {
	tq, ok := tickerQuery(w, r)
	if !ok {
		return
	}
	if tq.From.IsZero() {
		writeError(w, http.StatusBadRequest, "from is required")
		return
	}
	if s := r.URL.Query().Get("limit"); s != "" {
		var err error
		if tq.Limit, err = strconv.Atoi(s); err != nil || tq.Limit == 0 {
			writeError(w, http.StatusBadRequest, "limit must be an integer between 1 and 20")
			return
		}
	}
	items, err := h.svc.TopTickers(r.Context(), tq)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Cache-Control", publicCache)
	writeJSON(w, http.StatusOK, toTickerCounts(items))
}

// GET /api/v1/indices/{index}/instruments?from=&kind= — every member with its count.
func (h handlers) indexInstruments(w http.ResponseWriter, r *http.Request) {
	tq, ok := tickerQuery(w, r)
	if !ok {
		return
	}
	tq.Index = r.PathValue("index")
	items, err := h.svc.IndexMembers(r.Context(), tq)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Cache-Control", publicCache)
	writeJSON(w, http.StatusOK, toTickerCounts(items))
}

// tickerQuery reads index, kind and from; on bad input it writes 400 and returns false.
func tickerQuery(w http.ResponseWriter, r *http.Request) (service.TickerQuery, bool) {
	q := r.URL.Query()
	tq := service.TickerQuery{Index: q.Get("index")}
	if k := q.Get("kind"); k != "" {
		tq.Kinds = strings.Split(k, ",")
	}
	from, err := optTime(q.Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "from must be RFC 3339")
		return tq, false
	}
	if from != nil {
		tq.From = *from
	}
	return tq, true
}

func toTickerCounts(items []service.TickerCount) tickerCountListDTO {
	out := tickerCountListDTO{Items: make([]tickerCountDTO, len(items))}
	for i, it := range items {
		out.Items[i] = tickerCountDTO{Ticker: it.Ticker, Name: it.Name, Count: it.Mentions}
	}
	return out
}
