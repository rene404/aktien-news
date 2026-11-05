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
	// Every stored article sits in a cluster; alone until w.cluster merges it.
	if _, err := w.pool.Exec(ctx, `
		WITH c AS (INSERT INTO clusters (first_seen_at, representative_article_id) VALUES (now(), $1) RETURNING id)
		UPDATE articles SET cluster_id = (SELECT id FROM c) WHERE id = $1`, a.ID); err != nil {
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

// cluster moves the members into the representative's cluster.
func (w *world) cluster(t *testing.T, representative int64, members ...int64) {
	t.Helper()
	ctx := context.Background()
	for _, m := range members {
		if _, err := w.pool.Exec(ctx, `
			UPDATE articles SET cluster_id = (SELECT cluster_id FROM articles WHERE id = $1) WHERE id = $2`,
			representative, m); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(ctx, `
		UPDATE clusters c SET size = (SELECT count(*) FROM articles a WHERE a.cluster_id = c.id)
		WHERE c.id = (SELECT cluster_id FROM articles WHERE id = $1)`, representative); err != nil {
		t.Fatal(err)
	}
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

func TestListNewsOneItemPerCluster(t *testing.T) {
	w := newWorld(t)
	rep := w.article(t, w.adhoc, 2, "EQS-Adhoc: Erste Group Bank AG: Dividende", link{w.erste, "linked"})
	dup := w.article(t, w.other, 1, "EQS-Adhoc: Erste Group Bank AG: Dividende (FN USA)", link{w.post, "linked"})
	w.article(t, w.adhoc, 3, "alone", link{w.apple, "linked"})
	w.cluster(t, rep, dup)

	page, err := w.svc.ListNews(context.Background(), service.NewsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := titles(page), []string{"alone", "EQS-Adhoc: Erste Group Bank AG: Dividende"}; !slices.Equal(got, want) {
		t.Fatalf("titles = %v, want %v", got, want)
	}
	// The representative's title, link and source; the tickers of every member.
	it := page.Items[1]
	if it.ID != rep || it.SourceKey != "fn_adhoc" || !slices.Equal(it.Tickers, []string{"EBS", "POST"}) {
		t.Errorf("cluster item = %+v", it)
	}
}

// "Auch bei": the other sources of a cluster, one member each (the earliest),
// never the representative's own source, oldest first.
func TestListNewsAlsoAt(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	third, err := w.q.UpsertSource(ctx, db.UpsertSourceParams{Key: "ots_finanzen", Name: "APA-OTS Finanzen", Kind: db.SourceKindPressRelease, Market: "AT", Url: "https://example.com/ots", Parser: "ots"})
	if err != nil {
		t.Fatal(err)
	}
	rep := w.article(t, w.adhoc, 10, "Erste: Dividende", link{w.erste, "linked"})
	sameSource := w.article(t, w.adhoc, 11, "Erste: Dividende (again)")
	usaLate := w.article(t, w.other, 9, "Erste dividend, later copy")
	usaEarly := w.article(t, w.other, 5, "Erste dividend")
	ots := w.article(t, third.ID, 7, "Erste Group: Dividende beschlossen")
	w.cluster(t, rep, sameSource, usaLate, usaEarly, ots)
	w.article(t, w.adhoc, 20, "alone", link{w.apple, "linked"})

	page, err := w.svc.ListNews(ctx, service.NewsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(page.Items[0].Also); n != 0 {
		t.Errorf("alone: also = %+v, want none", page.Items[0].Also)
	}
	also := page.Items[1].Also
	var got []string
	for _, a := range also {
		got = append(got, a.SourceKey+": "+a.Title)
	}
	want := []string{"fn_usa: Erste dividend", "ots_finanzen: Erste Group: Dividende beschlossen"}
	if !slices.Equal(got, want) {
		t.Fatalf("also = %v, want %v", got, want)
	}
	if a := also[1]; a.ID != ots || a.SourceName != "APA-OTS Finanzen" || a.URL == "" || a.PublishedAt.IsZero() {
		t.Errorf("also[1] = %+v", a)
	}
}

func TestListNewsAlsoAtAtMostFive(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	rep := w.article(t, w.adhoc, 30, "rep", link{w.erste, "linked"})
	var members []int64
	for i := range 7 {
		src, err := w.q.UpsertSource(ctx, db.UpsertSourceParams{Key: fmt.Sprintf("src_%d", i), Name: fmt.Sprintf("Source %d", i), Kind: db.SourceKindNewsApi, Market: "AT", Url: fmt.Sprintf("https://example.com/s%d", i), Parser: "x"})
		if err != nil {
			t.Fatal(err)
		}
		members = append(members, w.article(t, src.ID, i, fmt.Sprintf("copy %d", i)))
	}
	w.cluster(t, rep, members...)
	page, err := w.svc.ListNews(ctx, service.NewsQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(page.Items[0].Also); n != service.MaxAlso {
		t.Errorf("also has %d entries, want %d", n, service.MaxAlso)
	}
}

func TestListNewsClusterFilters(t *testing.T) {
	w := newWorld(t)
	// Only the non-representative member is linked, and to Post.
	rep := w.article(t, w.adhoc, 2, "rep", link{w.erste, "pending"})
	member := w.article(t, w.other, 1, "member", link{w.post, "linked"})
	w.cluster(t, rep, member)
	ctx := context.Background()

	cases := []struct {
		name string
		q    service.NewsQuery
		want []string
	}{
		{"ticker of a member", service.NewsQuery{Ticker: "POST"}, []string{"rep"}},
		{"index of a member", service.NewsQuery{Index: "ATX"}, []string{"rep"}},
		{"pending link does not count", service.NewsQuery{Ticker: "EBS"}, []string{}},
		{"source is the representative's", service.NewsQuery{Source: "fn_adhoc"}, []string{"rep"}},
		{"member's source does not match", service.NewsQuery{Source: "fn_usa"}, []string{}},
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
		{"kind", service.NewsQuery{Kinds: []string{"news_api"}}, []string{"apple"}},
		{"several kinds", service.NewsQuery{Kinds: []string{"adhoc", "filing"}}, []string{"post", "erste"}},
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
	var ids []int64
	for i := range 5 {
		ids = append(ids, w.article(t, w.adhoc, i, fmt.Sprintf("a%d", i), link{w.erste, "linked"}))
	}
	// Same timestamp: the id breaks the tie, nothing is skipped or repeated.
	w.article(t, w.adhoc, 2, "a2-twin", link{w.erste, "linked"})
	// Cluster members never show up as items of their own.
	for i := range 3 {
		m := w.article(t, w.other, i, fmt.Sprintf("a%d-copy", i), link{w.erste, "linked"})
		w.cluster(t, ids[i], m)
	}

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
		"unknown kind":    {Kinds: []string{"gossip"}},
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

// Regression: with thousands of SEC issuers, a name substring matched
// alphabetically pushed the index stocks out of the ten suggestions.
func TestSearchInstrumentsRanking(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	for i := range 12 {
		if _, err := w.pool.Exec(ctx, `
			WITH c AS (INSERT INTO companies (slug, name, country) VALUES ($2, $1, 'US') RETURNING id)
			INSERT INTO listings (company_id, ticker, mic) SELECT id, $2, 'XNYS' FROM c`,
			fmt.Sprintf("A Flowers %02d Inc", i), fmt.Sprintf("FL%02d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(ctx, `
		WITH c AS (INSERT INTO companies (slug, name, country) VALUES ('ersatzteil', 'Ersatzteil Corp', 'US') RETURNING id)
		INSERT INTO listings (company_id, ticker, mic) SELECT id, 'ERSZ', 'XNYS' FROM c`); err != nil {
		t.Fatal(err)
	}

	got, err := w.svc.SearchInstruments(ctx, "ers")
	if err != nil {
		t.Fatal(err)
	}
	var tickers []string
	for _, g := range got {
		tickers = append(tickers, g.Ticker)
	}
	// Name prefix first, index members before the rest; then the substring hits.
	if len(tickers) != 10 || tickers[0] != "EBS" || tickers[1] != "ERSZ" {
		t.Fatalf("search ers = %v, want EBS, ERSZ, then substring hits", tickers)
	}

	// An exact ticker still wins over everything.
	got, _ = w.svc.SearchInstruments(ctx, "fl03")
	if len(got) == 0 || got[0].Ticker != "FL03" {
		t.Fatalf("search fl03 = %+v", got)
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

// last_ok_at: the end of the newest run that brought data (ok or partial);
// failed and skipped runs, and runs still going, do not count.
func TestSourcesLastOK(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	run := func(source int64, finished *time.Time, status string) {
		t.Helper()
		if _, err := w.pool.Exec(ctx, `INSERT INTO ingest_runs (source_id, started_at, finished_at, status)
			VALUES ($1, $2, $3, $4::run_status)`, source, finished.Add(-30*time.Second), finished, status); err != nil {
			t.Fatal(err)
		}
	}
	at := func(min int) *time.Time { v := w.base.Add(time.Duration(min) * time.Minute); return &v }
	run(w.adhoc, at(1), "ok")
	run(w.adhoc, at(2), "partial")
	run(w.adhoc, at(3), "failed")
	run(w.adhoc, at(4), "skipped")
	if _, err := w.pool.Exec(ctx, `INSERT INTO ingest_runs (source_id, started_at) VALUES ($1, $2)`, w.adhoc, *at(5)); err != nil {
		t.Fatal(err) // still running: no finished_at, no status
	}
	run(w.other, at(1), "failed")

	got, err := w.svc.Sources(ctx)
	if err != nil {
		t.Fatal(err)
	}
	last := map[string]*time.Time{}
	for _, s := range got {
		last[s.Key] = s.LastOKAt
	}
	if a := last["fn_adhoc"]; a == nil || !a.Equal(*at(2)) {
		t.Errorf("fn_adhoc last ok = %v, want %v", a, *at(2))
	}
	if u, ok := last["fn_usa"]; !ok || u != nil {
		t.Errorf("fn_usa last ok = %v (listed %v), want nil", u, ok)
	}
}
