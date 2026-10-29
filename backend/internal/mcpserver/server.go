// Package mcpserver exposes the internal news tools (search_news,
// ticker_sentiment) to mcp-hub over MCP Streamable HTTP. The tools are thin:
// parse arguments, call service, shape the result. Unlike the public REST
// API they return sentiment and event type, so the handler sits behind a
// bearer token and must never be reachable from the internet.
package mcpserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// Handler serves MCP at any path. It is stateless and answers with plain
// JSON (no SSE), which is what the hub proxy and McpHubClient expect.
func Handler(svc *service.Service, token string) http.Handler {
	server := NewServer(svc)
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	return requireToken(token, h)
}

func NewServer(svc *service.Service) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "aktien-news", Version: "0.3.0"}, nil)
	t := tools{svc: svc}
	mcp.AddTool(s, &mcp.Tool{
		Name: "search_news",
		Description: "News articles linked to companies, newest first, one item per (article, company). " +
			"Point-in-time: with as_of only what was known then counts. Classification fields are null until classified.",
	}, t.searchNews)
	mcp.AddTool(s, &mcp.Tool{
		Name: "ticker_sentiment",
		Description: "Sentiment of one ticker's linked articles in a window ending at as_of. " +
			"score_weighted = Σ(sentiment·confidence·source_weight·novelty) / Σ(confidence·source_weight·novelty); scores are null without classified articles.",
	}, t.tickerSentiment)
	return s
}

func requireToken(token string, next http.Handler) http.Handler {
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="aktien-news"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type tools struct{ svc *service.Service }

type SearchNewsArgs struct {
	Ticker        string   `json:"ticker,omitempty" jsonschema:"ticker symbol, e.g. EBS or AAPL"`
	Query         string   `json:"query,omitempty" jsonschema:"case-insensitive substring of the title"`
	Since         string   `json:"since" jsonschema:"RFC 3339 start of the publication window (inclusive)"`
	Until         string   `json:"until,omitempty" jsonschema:"RFC 3339 end of the publication window (exclusive)"`
	EventType     string   `json:"event_type,omitempty" jsonschema:"earnings, guidance, m_and_a, rating_change, litigation, insider, management_change, capital_markets, dividend, regulatory, adhoc_other or other"`
	MinConfidence *float64 `json:"min_confidence,omitempty" jsonschema:"minimum company-link confidence, default 0.85"`
	AsOf          string   `json:"as_of,omitempty" jsonschema:"RFC 3339 knowledge time, default now"`
	Limit         int      `json:"limit,omitempty" jsonschema:"1-50, default 20"`
}

type SearchNewsResult struct {
	Items []NewsItem `json:"items"`
}

type NewsItem struct {
	ID             int64      `json:"id"`
	ClusterID      int64      `json:"cluster_id"`
	Title          string     `json:"title"`
	URL            string     `json:"url"`
	Source         Source     `json:"source"`
	SourceWeight   float64    `json:"source_weight"`
	PublishedAt    time.Time  `json:"published_at"`
	IngestedAt     time.Time  `json:"ingested_at"`
	Company        string     `json:"company"`
	Tickers        []string   `json:"tickers"`
	LinkConfidence float64    `json:"link_confidence"`
	EventType      *string    `json:"event_type"`
	Sentiment      *float64   `json:"sentiment"`
	Confidence     *float64   `json:"confidence"`
	Novelty        *float64   `json:"novelty"`
	ClassifiedAt   *time.Time `json:"classified_at"`
}

type Source struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

func (t tools) searchNews(ctx context.Context, _ *mcp.CallToolRequest, a SearchNewsArgs) (*mcp.CallToolResult, SearchNewsResult, error) {
	q := service.SearchQuery{Ticker: a.Ticker, Query: a.Query, EventType: a.EventType, MinConfidence: a.MinConfidence, Limit: a.Limit}
	var err error
	if a.Since == "" {
		return nil, SearchNewsResult{}, errors.New("since is required (RFC 3339)")
	}
	if q.Since, err = parseTime("since", a.Since); err != nil {
		return nil, SearchNewsResult{}, err
	}
	if q.Until, err = parseOptionalTime("until", a.Until); err != nil {
		return nil, SearchNewsResult{}, err
	}
	if q.AsOf, err = parseOptionalTime("as_of", a.AsOf); err != nil {
		return nil, SearchNewsResult{}, err
	}

	items, err := t.svc.SearchNews(ctx, q)
	if err != nil {
		return nil, SearchNewsResult{}, toolError("search_news", err)
	}
	out := SearchNewsResult{Items: make([]NewsItem, len(items))}
	for i, it := range items {
		n := NewsItem{
			ID: it.ID, ClusterID: it.ClusterID, Title: it.Title, URL: it.URL,
			Source: Source{Key: it.SourceKey, Name: it.SourceName}, SourceWeight: it.SourceWeight,
			PublishedAt: it.PublishedAt, IngestedAt: it.IngestedAt,
			Company: it.Company, Tickers: it.Tickers, LinkConfidence: it.LinkConfidence,
		}
		if c := it.Classification; c != nil {
			n.EventType, n.Sentiment, n.Confidence, n.Novelty, n.ClassifiedAt = &c.EventType, &c.Sentiment, &c.Confidence, &c.Novelty, &c.ClassifiedAt
		}
		out.Items[i] = n
	}
	return nil, out, nil
}

type TickerSentimentArgs struct {
	Ticker string `json:"ticker" jsonschema:"ticker symbol, e.g. EBS or AAPL"`
	Window string `json:"window,omitempty" jsonschema:"1d, 7d or 30d, default 7d"`
	AsOf   string `json:"as_of,omitempty" jsonschema:"RFC 3339 knowledge time and window end, default now"`
}

type SentimentResult struct {
	Ticker        string                `json:"ticker"`
	Company       string                `json:"company"`
	Window        string                `json:"window"`
	AsOf          time.Time             `json:"as_of"`
	NArticles     int                   `json:"n_articles"`
	NClusters     int                   `json:"n_clusters"`
	NClassified   int                   `json:"n_classified"`
	ScoreWeighted *float64              `json:"score_weighted"`
	ScoreMean     *float64              `json:"score_mean"`
	ByEventType   map[string]EventScore `json:"by_event_type"`
	// Baseline is an external provider's score; null until one is licensed.
	Baseline any `json:"baseline"`
}

type EventScore struct {
	N     int      `json:"n"`
	Score *float64 `json:"score"`
}

func (t tools) tickerSentiment(ctx context.Context, _ *mcp.CallToolRequest, a TickerSentimentArgs) (*mcp.CallToolResult, SentimentResult, error) {
	asOf, err := parseOptionalTime("as_of", a.AsOf)
	if err != nil {
		return nil, SentimentResult{}, err
	}
	s, err := t.svc.TickerSentiment(ctx, service.SentimentQuery{Ticker: a.Ticker, Window: a.Window, AsOf: asOf})
	if errors.Is(err, service.ErrNotFound) {
		return nil, SentimentResult{}, fmt.Errorf("unknown ticker %q", a.Ticker)
	}
	if err != nil {
		return nil, SentimentResult{}, toolError("ticker_sentiment", err)
	}
	out := SentimentResult{
		Ticker: s.Ticker, Company: s.Company, Window: s.Window, AsOf: s.AsOf,
		NArticles: s.NArticles, NClusters: s.NClusters, NClassified: s.NClassified,
		ScoreWeighted: s.ScoreWeighted, ScoreMean: s.ScoreMean,
		ByEventType: make(map[string]EventScore, len(s.ByEventType)),
	}
	for k, v := range s.ByEventType {
		out.ByEventType[k] = EventScore{N: v.N, Score: v.Score}
	}
	return nil, out, nil
}

// toolError passes validation messages to the caller (isError) and hides
// everything else behind a generic message.
func toolError(tool string, err error) error {
	var ve *service.ValidationError
	if errors.As(err, &ve) {
		return ve
	}
	slog.Error("mcp tool failed", "tool", tool, "err", err)
	return errors.New("internal error")
}

func parseTime(name, v string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC 3339, e.g. 2026-10-29T12:00:00Z", name)
	}
	return t, nil
}

func parseOptionalTime(name, v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := parseTime(name, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
