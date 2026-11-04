package service

import (
	"context"
	"slices"
	"time"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

const (
	DefaultTopLimit = 10
	MaxTopLimit     = 20
)

// TickerQuery selects what the ticker counts count; the same filters as the
// news feed (index, source kinds, from).
type TickerQuery struct {
	Index string
	Kinds []string // empty = all
	From  time.Time
	Limit int // TopTickers only; 0 = DefaultTopLimit
}

// TickerCount is a ticker with the number of news clusters linked to it.
type TickerCount struct {
	Ticker   string
	Name     string
	Mentions int
}

// TopTickers returns the most-mentioned tickers since q.From, most first.
func (s *Service) TopTickers(ctx context.Context, q TickerQuery) ([]TickerCount, error) {
	limit := q.Limit
	if limit == 0 {
		limit = DefaultTopLimit
	}
	if limit < 1 || limit > MaxTopLimit {
		return nil, invalid("limit must be between 1 and %d", MaxTopLimit)
	}
	if q.From.IsZero() {
		return nil, invalid("from is required")
	}
	if err := validateTickerQuery(q); err != nil {
		return nil, err
	}
	rows, err := s.q.TopTickers(ctx, db.TopTickersParams{
		IndexKey: optional(q.Index), Kinds: q.Kinds, FromTs: q.From, Lim: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]TickerCount, len(rows))
	for i, r := range rows {
		out[i] = TickerCount{Ticker: r.Ticker, Name: r.Name, Mentions: int(r.Mentions)}
	}
	return out, nil
}

// IndexMembers returns every current member of q.Index with its count since
// q.From (all time when zero), sorted by name. ErrNotFound when the index
// has no current members.
func (s *Service) IndexMembers(ctx context.Context, q TickerQuery) ([]TickerCount, error) {
	if q.Index == "" {
		return nil, invalid("index is required")
	}
	if err := validateTickerQuery(q); err != nil {
		return nil, err
	}
	params := db.ListIndexMembersParams{IndexKey: q.Index, Kinds: q.Kinds}
	if !q.From.IsZero() {
		params.FromTs = &q.From
	}
	rows, err := s.q.ListIndexMembers(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	out := make([]TickerCount, len(rows))
	for i, r := range rows {
		out[i] = TickerCount{Ticker: r.Ticker, Name: r.Name, Mentions: int(r.Mentions)}
	}
	return out, nil
}

func validateTickerQuery(q TickerQuery) error {
	if q.Index != "" && !keyPattern.MatchString(q.Index) {
		return invalid("invalid index")
	}
	for _, k := range q.Kinds {
		if !slices.Contains(SourceKinds, k) {
			return invalid("unknown source kind %q", k)
		}
	}
	return nil
}
