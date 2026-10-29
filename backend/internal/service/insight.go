package service

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

// Internal (MCP-only) queries: they expose classifications, so no public
// handler may call them. Both are point-in-time: only what was known at
// AsOf counts (design §6).

const (
	DefaultSearchLimit   = 20
	MaxSearchLimit       = 50
	DefaultMinConfidence = 0.85
	maxQueryLen          = 200
	DefaultWindow        = "7d"
)

// EventTypes are the values of the event_type enum.
var EventTypes = []string{
	"earnings", "guidance", "m_and_a", "rating_change", "litigation", "insider",
	"management_change", "capital_markets", "dividend", "regulatory", "adhoc_other", "other",
}

// Windows are the look-back periods of TickerSentiment.
var Windows = map[string]time.Duration{
	"1d":  24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

type SearchQuery struct {
	Ticker        string
	Query         string // case-insensitive substring of the title
	Since         time.Time
	Until         *time.Time // exclusive
	EventType     string
	MinConfidence *float64   // link confidence; nil = DefaultMinConfidence
	AsOf          *time.Time // nil = now
	Limit         int        // 0 = DefaultSearchLimit
}

// Insight is one article linked to one company, with that pair's newest
// classification at AsOf (nil if there was none).
type Insight struct {
	ID             int64
	ClusterID      int64
	Title          string
	URL            string
	SourceKey      string
	SourceName     string
	SourceWeight   float64
	PublishedAt    time.Time
	IngestedAt     time.Time
	Company        string
	Tickers        []string
	LinkConfidence float64
	Classification *Classification
}

type Classification struct {
	EventType    string
	Sentiment    float64
	Confidence   float64
	Novelty      float64
	ClassifiedAt time.Time
}

func (s *Service) SearchNews(ctx context.Context, sq SearchQuery) ([]Insight, error) {
	limit := sq.Limit
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	minConf := DefaultMinConfidence
	if sq.MinConfidence != nil {
		minConf = *sq.MinConfidence
	}
	switch {
	case sq.Since.IsZero():
		return nil, invalid("since is required")
	case sq.Until != nil && !sq.Since.Before(*sq.Until):
		return nil, invalid("since must be before until")
	case limit < 1 || limit > MaxSearchLimit:
		return nil, invalid("limit must be between 1 and %d", MaxSearchLimit)
	case minConf < 0 || minConf > 1:
		return nil, invalid("min_confidence must be between 0 and 1")
	case sq.Ticker != "" && !tickerPattern.MatchString(sq.Ticker):
		return nil, invalid("invalid ticker")
	case sq.EventType != "" && !slices.Contains(EventTypes, sq.EventType):
		return nil, invalid("unknown event type %q", sq.EventType)
	case len(sq.Query) > maxQueryLen:
		return nil, invalid("query must be at most %d bytes", maxQueryLen)
	}

	params := db.SearchInsightsParams{
		AsOf: s.asOf(sq.AsOf), Since: sq.Since, Until: sq.Until,
		MinConfidence: minConf, Ticker: optional(sq.Ticker), EventType: optional(sq.EventType),
		Lim: int32(limit),
	}
	if sq.Query != "" {
		pattern := "%" + likeEscaper.Replace(sq.Query) + "%"
		params.TitlePattern = &pattern
	}
	rows, err := s.q.SearchInsights(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]Insight, len(rows))
	for i, r := range rows {
		out[i] = Insight{
			ID: r.ID, ClusterID: deref(r.ClusterID), Title: r.Title, URL: r.Url,
			SourceKey: r.SourceKey, SourceName: r.SourceName, SourceWeight: r.SourceWeight,
			PublishedAt: r.PublishedAt.UTC(), IngestedAt: r.IngestedAt.UTC(),
			Company: r.CompanyName, Tickers: r.Tickers, LinkConfidence: r.LinkConfidence,
		}
		if r.EventType != nil {
			out[i].Classification = &Classification{
				EventType: *r.EventType, Sentiment: deref(r.Sentiment), Confidence: deref(r.Confidence),
				Novelty: deref(r.Novelty), ClassifiedAt: deref(r.ClassifiedAt).UTC(),
			}
		}
	}
	return out, nil
}

// likeEscaper makes a user string literal inside an ILIKE pattern.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

type SentimentQuery struct {
	Ticker string
	Window string     // "" = DefaultWindow
	AsOf   *time.Time // nil = now
}

// Sentiment aggregates a company's linked articles published in
// (AsOf − Window, AsOf]. Scores cover only classified articles and are nil
// when there are none.
type Sentiment struct {
	Ticker        string
	Company       string
	Window        string
	AsOf          time.Time
	NArticles     int
	NClusters     int
	NClassified   int
	ScoreWeighted *float64
	ScoreMean     *float64
	ByEventType   map[string]EventScore
}

type EventScore struct {
	N     int
	Score *float64 // weighted like ScoreWeighted
}

func (s *Service) TickerSentiment(ctx context.Context, sq SentimentQuery) (Sentiment, error) {
	window := sq.Window
	if window == "" {
		window = DefaultWindow
	}
	span, ok := Windows[window]
	switch {
	case sq.Ticker == "":
		return Sentiment{}, invalid("ticker is required")
	case !tickerPattern.MatchString(sq.Ticker):
		return Sentiment{}, invalid("invalid ticker")
	case !ok:
		return Sentiment{}, invalid("window must be 1d, 7d or 30d")
	}

	company, err := s.q.CompanyForTicker(ctx, sq.Ticker)
	if err != nil {
		return Sentiment{}, notFound(err)
	}
	asOf := s.asOf(sq.AsOf)
	rows, err := s.q.SentimentRows(ctx, db.SentimentRowsParams{CompanyID: company.ID, AsOf: asOf, FromTs: asOf.Add(-span)})
	if err != nil {
		return Sentiment{}, err
	}

	out := Sentiment{
		Ticker: strings.ToUpper(sq.Ticker), Company: company.Name, Window: window, AsOf: asOf,
		NArticles: len(rows), ByEventType: map[string]EventScore{},
	}
	clusters := map[int64]bool{}
	var all score
	byType := map[string]*score{}
	for _, r := range rows {
		clusters[deref(r.ClusterID)] = true
		if r.EventType == nil {
			continue
		}
		weight := deref(r.Confidence) * r.SourceWeight * deref(r.Novelty)
		all.add(deref(r.Sentiment), weight)
		if byType[*r.EventType] == nil {
			byType[*r.EventType] = &score{}
		}
		byType[*r.EventType].add(deref(r.Sentiment), weight)
	}
	out.NClusters = len(clusters)
	out.NClassified = all.n
	out.ScoreWeighted, out.ScoreMean = all.weighted(), all.mean()
	for t, sc := range byType {
		out.ByEventType[t] = EventScore{N: sc.n, Score: sc.weighted()}
	}
	return out, nil
}

// score accumulates Σ(sentiment·weight)/Σ(weight), weight =
// confidence·source_weight·novelty, and the plain mean.
type score struct {
	n                 int
	sum, wsum, weight float64
}

func (s *score) add(sentiment, weight float64) {
	s.n++
	s.sum += sentiment
	s.wsum += sentiment * weight
	s.weight += weight
}

func (s *score) weighted() *float64 {
	if s.weight == 0 {
		return nil
	}
	v := s.wsum / s.weight
	return &v
}

func (s *score) mean() *float64 {
	if s.n == 0 {
		return nil
	}
	v := s.sum / float64(s.n)
	return &v
}

func (s *Service) asOf(t *time.Time) time.Time {
	if t != nil {
		return t.UTC()
	}
	return time.Now().UTC()
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}
