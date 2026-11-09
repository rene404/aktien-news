// Package service holds the read-side query logic shared by the public REST
// API and (Phase 3) the MCP server. It validates input and returns plain
// service types; transports map them to their own DTOs.
package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

const (
	DefaultLimit = 25
	MaxLimit     = 100
	// MaxAlso caps "Auch bei": the other sources listed per news item.
	MaxAlso = 5
)

var ErrNotFound = errors.New("not found")

// SourceKinds are the values of the source_kind enum.
var SourceKinds = []string{"adhoc", "press_release", "news_api", "ir", "filing"}

// ValidationError means the caller sent bad input (HTTP 400).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

type Service struct {
	q *db.Queries
}

func New(conn db.DBTX) *Service {
	return &Service{q: db.New(conn)}
}

type NewsQuery struct {
	Ticker   string
	Index    string
	Source   string
	Kinds    []string // source kinds; empty = all
	From, To *time.Time
	Cursor   string
	Limit    int // 0 = DefaultLimit
}

type NewsItem struct {
	ID          int64
	Title       string
	URL         string
	SourceKey   string
	SourceName  string
	SourceKind  string
	PublishedAt time.Time
	Tickers     []string
	// The same news at other sources (same cluster), oldest first; at most MaxAlso.
	Also []AlsoAt
	// SameSource counts the further items of the item's own source in its
	// cluster, e.g. several directors' dealings of one issuer.
	SameSource int
}

// AlsoAt is one other source's copy of a news item.
type AlsoAt struct {
	ID          int64
	Title       string
	URL         string
	SourceKey   string
	SourceName  string
	SourceKind  string
	PublishedAt time.Time
}

type NewsPage struct {
	Items      []NewsItem
	NextCursor string // "" on the last page
}

var (
	tickerPattern = regexp.MustCompile(`^[A-Za-z0-9.\-]{1,12}$`)
	keyPattern    = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)
)

func (s *Service) ListNews(ctx context.Context, nq NewsQuery) (NewsPage, error) {
	limit := nq.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit < 1 || limit > MaxLimit {
		return NewsPage{}, invalid("limit must be between 1 and %d", MaxLimit)
	}
	if nq.From != nil && nq.To != nil && !nq.From.Before(*nq.To) {
		return NewsPage{}, invalid("from must be before to")
	}
	if nq.Ticker != "" && !tickerPattern.MatchString(nq.Ticker) {
		return NewsPage{}, invalid("invalid ticker")
	}
	if nq.Index != "" && !keyPattern.MatchString(nq.Index) {
		return NewsPage{}, invalid("invalid index")
	}
	if nq.Source != "" && !keyPattern.MatchString(nq.Source) {
		return NewsPage{}, invalid("invalid source")
	}
	for _, k := range nq.Kinds {
		if !slices.Contains(SourceKinds, k) {
			return NewsPage{}, invalid("unknown source kind %q", k)
		}
	}

	params := db.ListNewsParams{
		Source: optional(nq.Source), Ticker: optional(nq.Ticker), IndexKey: optional(nq.Index),
		Kinds:  nq.Kinds,
		FromTs: nq.From, ToTs: nq.To,
		Lim: int32(limit + 1), // one extra row tells us whether there is a next page
	}
	if nq.Cursor != "" {
		ts, id, err := decodeCursor(nq.Cursor)
		if err != nil {
			return NewsPage{}, invalid("invalid cursor")
		}
		params.CursorTs, params.CursorID = &ts, &id
	}

	rows, err := s.q.ListNews(ctx, params)
	if err != nil {
		return NewsPage{}, err
	}
	page := NewsPage{Items: make([]NewsItem, 0, min(len(rows), limit))}
	for i, r := range rows {
		if i == limit {
			last := page.Items[limit-1]
			page.NextCursor = encodeCursor(last.PublishedAt, last.ID)
			break
		}
		page.Items = append(page.Items, NewsItem{
			ID: r.ID, Title: r.Title, URL: r.Url, SourceKey: r.SourceKey, SourceName: r.SourceName, SourceKind: string(r.SourceKind),
			PublishedAt: r.PublishedAt.UTC(), Tickers: r.Tickers,
		})
	}
	if err := s.attachAlso(ctx, page.Items); err != nil {
		return NewsPage{}, err
	}
	return page, nil
}

// attachAlso fills Also and SameSource for the items of one page, one query each.
func (s *Service) attachAlso(ctx context.Context, items []NewsItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	rows, err := s.q.ListAlsoAt(ctx, ids)
	if err != nil {
		return err
	}
	byRep := map[int64][]AlsoAt{}
	for _, r := range rows {
		byRep[r.RepresentativeID] = append(byRep[r.RepresentativeID], AlsoAt{
			ID: r.ID, Title: r.Title, URL: r.Url, SourceKey: r.SourceKey, SourceName: r.SourceName,
			SourceKind: string(r.SourceKind), PublishedAt: r.PublishedAt.UTC(),
		})
	}
	counts, err := s.q.CountSameSource(ctx, ids)
	if err != nil {
		return err
	}
	same := make(map[int64]int, len(counts))
	for _, c := range counts {
		same[c.RepresentativeID] = int(c.N)
	}
	for i := range items {
		items[i].SameSource = same[items[i].ID]
		also := byRep[items[i].ID]
		slices.SortFunc(also, func(a, b AlsoAt) int {
			if c := a.PublishedAt.Compare(b.PublishedAt); c != 0 {
				return c
			}
			return int(a.ID - b.ID)
		})
		if len(also) > MaxAlso {
			also = also[:MaxAlso]
		}
		items[i].Also = also
	}
	return nil
}

// The cursor is opaque to clients: base64("<unix nanos>:<id>").
func encodeCursor(ts time.Time, id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(ts.UnixNano(), 10) + ":" + strconv.FormatInt(id, 10)))
}

func decodeCursor(c string) (time.Time, int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, 0, err
	}
	tsPart, idPart, ok := strings.Cut(string(raw), ":")
	if !ok {
		return time.Time{}, 0, errors.New("malformed cursor")
	}
	nanos, err := strconv.ParseInt(tsPart, 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return time.Time{}, 0, err
	}
	return time.Unix(0, nanos).UTC(), id, nil
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
