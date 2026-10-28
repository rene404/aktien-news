package service_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/service"
	"github.com/rene404/aktien-news/backend/internal/store/db"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
)

// world: catalog testdata (Erste EBS/ATX, Post POST/ATX, Apple AAPL/SP500,
// source fn_adhoc) plus a second source.
type world struct {
	pool               *pgxpool.Pool
	q                  *db.Queries
	adhoc, other       int64
	erste, post, apple int64
	base               time.Time
	svc                *service.Service
}

func newWorld(t *testing.T) *world {
	t.Helper()
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("../catalog/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	adhoc, err := q.GetSourceByKey(ctx, "fn_adhoc")
	if err != nil {
		t.Fatal(err)
	}
	other, err := q.UpsertSource(ctx, db.UpsertSourceParams{Key: "fn_usa", Name: "FN USA", Kind: db.SourceKindNewsApi, Market: "US", Url: "https://example.com/usa", Parser: "finanznachrichten"})
	if err != nil {
		t.Fatal(err)
	}
	cat := catalog.New(pool)
	id := func(isin string) int64 {
		c, err := cat.CompanyByISIN(ctx, isin)
		if err != nil {
			t.Fatal(err)
		}
		return c.ID
	}
	return &world{
		pool: pool, q: q, adhoc: adhoc.ID, other: other.ID,
		erste: id("AT0000652011"), post: id("AT0000APOST4"), apple: id("US0378331005"),
		base: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC),
		svc:  service.New(pool),
	}
}

// article inserts an article published base+minutes ago-offset and records
// link rows (in order) as company:status pairs.
func (w *world) article(t *testing.T, source int64, minutes int, title string, links ...link) int64 {
	t.Helper()
	ctx := context.Background()
	url := fmt.Sprintf("https://example.com/%d-%s", minutes, title)
	a, err := w.q.InsertArticle(ctx, db.InsertArticleParams{
		SourceID: source, ExternalID: url, Url: url, CanonicalUrl: url, Title: title,
		PublishedAt: w.base.Add(time.Duration(minutes) * time.Minute), ContentHash: []byte(title),
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, l := range links {
		if _, err := w.pool.Exec(ctx, `
			INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by, decided_at)
			VALUES ($1, $2, 0.9, 'ticker', $3, 'engine', now() + make_interval(secs => $4))`,
			a.ID, l.company, l.status, i); err != nil {
			t.Fatal(err)
		}
	}
	return a.ID
}

type link struct {
	company int64
	status  string
}

func titles(p service.NewsPage) []string {
	out := make([]string, len(p.Items))
	for i, it := range p.Items {
		out[i] = it.Title
	}
	return out
}

func TestListNewsOnlyCurrentlyLinked(t *testing.T) {
	w := newWorld(t)
	w.article(t, w.adhoc, 1, "linked", link{w.erste, "linked"})
	w.article(t, w.adhoc, 2, "pending only", link{w.post, "pending"})
	w.article(t, w.adhoc, 3, "rejected later", link{w.post, "linked"}, link{w.post, "rejected"})
	w.article(t, w.adhoc, 4, "upgraded", link{w.apple, "pending"}, link{w.apple, "linked"})
	w.article(t, w.adhoc, 5, "unlinked")

	page, err := w.svc.ListNews(context.Background(), service.NewsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := titles(page), []string{"upgraded", "linked"}; !slices.Equal(got, want) {
		t.Fatalf("titles = %v, want %v", got, want)
	}
	first := page.Items[0]
	if first.SourceKey != "fn_adhoc" || first.URL == "" || !slices.Equal(first.Tickers, []string{"AAPL"}) {
		t.Errorf("item = %+v", first)
	}
}

func TestListNewsTickersOfAllLinkedCompanies(t *testing.T) {
	w := newWorld(t)
	w.article(t, w.adhoc, 1, "two", link{w.post, "linked"}, link{w.erste, "linked"}, link{w.apple, "pending"})
	page, err := w.svc.ListNews(context.Background(), service.NewsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || !slices.Equal(page.Items[0].Tickers, []string{"EBS", "POST"}) {
		t.Fatalf("items = %+v, want one item with [EBS POST]", page.Items)
	}
}

func TestListNewsFilters(t *testing.T) {
	w := newWorld(t)
	w.article(t, w.adhoc, 1, "erste", link{w.erste, "linked"})
	w.article(t, w.adhoc, 2, "post", link{w.post, "linked"})
	w.article(t, w.other, 3, "apple", link{w.apple, "linked"})
	w.article(t, w.adhoc, 4, "post pending", link{w.post, "pending"})
	ctx := context.Background()
	from := w.base.Add(2 * time.Minute)
	to := w.base.Add(3 * time.Minute)

	cases := []struct {
		name string
		q    service.NewsQuery
		want []string
	}{
		{"ticker", service.NewsQuery{Ticker: "POST"}, []string{"post"}},
		{"ticker case-insensitive", service.NewsQuery{Ticker: "ebs"}, []string{"erste"}},
		{"unknown ticker", service.NewsQuery{Ticker: "NOPE"}, []string{}},
		{"index", service.NewsQuery{Index: "ATX"}, []string{"post", "erste"}},
		{"index sp500", service.NewsQuery{Index: "SP500"}, []string{"apple"}},
		{"source", service.NewsQuery{Source: "fn_usa"}, []string{"apple"}},
		{"from/to inclusive from, exclusive to", service.NewsQuery{From: &from, To: &to}, []string{"post"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, err := w.svc.ListNews(ctx, tc.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := titles(page); !slices.Equal(got, tc.want) {
				t.Errorf("titles = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestListNewsPagination(t *testing.T) {
	w := newWorld(t)
	for i := range 5 {
		w.article(t, w.adhoc, i, fmt.Sprintf("a%d", i), link{w.erste, "linked"})
	}
	// Same timestamp: the id breaks the tie, nothing is skipped or repeated.
	w.article(t, w.adhoc, 2, "a2-twin", link{w.erste, "linked"})

	ctx := context.Background()
	var all []string
	cursor := ""
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("pagination does not terminate")
		}
		page, err := w.svc.ListNews(ctx, service.NewsQuery{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, titles(page)...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	want := []string{"a4", "a3", "a2-twin", "a2", "a1", "a0"}
	if !slices.Equal(all, want) {
		t.Fatalf("pages = %v, want %v", all, want)
	}
}

func TestListNewsValidation(t *testing.T) {
	w := newWorld(t)
	early := w.base
	late := w.base.Add(time.Hour)
	for name, q := range map[string]service.NewsQuery{
		"limit too small": {Limit: -1},
		"limit too large": {Limit: 101},
		"from after to":   {From: &late, To: &early},
		"bad cursor":      {Cursor: "not-a-cursor"},
		"bad ticker":      {Ticker: "DROP TABLE"},
		"bad index":       {Index: "atx; --"},
	} {
		_, err := w.svc.ListNews(context.Background(), q)
		var verr *service.ValidationError
		if !errors.As(err, &verr) {
			t.Errorf("%s: err = %v, want ValidationError", name, err)
		}
	}
}

func TestInstruments(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()

	got, err := w.svc.SearchInstruments(ctx, "erst")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Ticker != "EBS" || got[0].Name != "Erste Group" || got[0].ISIN != "AT0000652011" {
		t.Errorf("search erst = %+v", got)
	}
	got, err = w.svc.SearchInstruments(ctx, "aapl")
	if err != nil || len(got) != 1 || got[0].Name != "Apple" {
		t.Errorf("search aapl = %+v, %v", got, err)
	}
	got, err = w.svc.SearchInstruments(ctx, "AT0000APOST4")
	if err != nil || len(got) != 1 || got[0].Ticker != "POST" {
		t.Errorf("search by isin = %+v, %v", got, err)
	}
	if _, err := w.svc.SearchInstruments(ctx, ""); err == nil {
		t.Error("empty query: want error")
	}

	inst, err := w.svc.Instrument(ctx, "post")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Name != "Österreichische Post" || inst.MIC != "XWBO" || !slices.Equal(inst.Indices, []string{"ATX"}) {
		t.Errorf("instrument = %+v", inst)
	}
	if _, err := w.svc.Instrument(ctx, "NOPE"); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("unknown: err = %v, want ErrNotFound", err)
	}
}

func TestSources(t *testing.T) {
	w := newWorld(t)
	if _, err := w.pool.Exec(context.Background(), `UPDATE sources SET enabled = false WHERE key = 'fn_usa'`); err != nil {
		t.Fatal(err)
	}
	got, err := w.svc.Sources(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "fn_adhoc" || got[0].Market != "MULTI" {
		t.Errorf("sources = %+v", got)
	}
}
