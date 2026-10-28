package matching_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/matching"
	"github.com/rene404/aktien-news/backend/internal/store/db"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
)

type fixture struct {
	pool      *pgxpool.Pool
	articleID int64
	erste     int64
	post      int64
	apple     int64
}

func setup(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("../catalog/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	src, err := q.GetSourceByKey(ctx, "fn_adhoc")
	if err != nil {
		t.Fatal(err)
	}
	a, err := q.InsertArticle(ctx, db.InsertArticleParams{
		SourceID: src.ID, ExternalID: "x", Url: "https://example.com/x", CanonicalUrl: "https://example.com/x",
		Title: "t", PublishedAt: time.Now(), ContentHash: []byte{1},
	})
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
	return fixture{pool, a.ID, id("AT0000652011"), id("AT0000APOST4"), id("US0378331005")}
}

func current(t *testing.T, f fixture) map[int64]string {
	t.Helper()
	links, err := matching.CurrentLinks(context.Background(), f.pool, f.articleID)
	if err != nil {
		t.Fatal(err)
	}
	out := map[int64]string{}
	for _, l := range links {
		out[l.CompanyID] = l.Status
	}
	return out
}

func rowCount(t *testing.T, f fixture) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM article_entities").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLoadIndexMatchesSeededCompanies(t *testing.T) {
	f := setup(t)
	ix, err := matching.LoadIndex(context.Background(), f.pool)
	if err != nil {
		t.Fatal(err)
	}
	got := ix.Match(matching.Input{Title: "EQS-Adhoc: Erste Group Bank AG: Kapitalmaßnahme", IssuerHint: "Erste Group Bank AG"})
	if len(got) != 1 || got[0].CompanyID != f.erste || got[0].Method != "issuer" {
		t.Fatalf("got %+v, want erste via issuer", got)
	}
	got = ix.Match(matching.Input{Title: "Why $AAPL rallies"})
	if len(got) != 1 || got[0].CompanyID != f.apple {
		t.Fatalf("got %+v, want apple", got)
	}
}

func TestPersistIsIdempotent(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	ms := []matching.Match{
		{CompanyID: f.erste, Confidence: 1.0, Method: "issuer"},
		{CompanyID: f.post, Confidence: 0.55, Method: "core_name"},
	}
	for range 2 {
		if err := matching.Persist(ctx, f.pool, f.articleID, ms); err != nil {
			t.Fatal(err)
		}
	}
	if n := rowCount(t, f); n != 2 {
		t.Fatalf("%d rows, want 2", n)
	}
	if got := current(t, f); got[f.erste] != "linked" || got[f.post] != "pending" {
		t.Fatalf("current = %v", got)
	}
}

func TestPersistUpgradesPendingToLinkedButNeverDowngrades(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	persist := func(conf float64, method string) {
		t.Helper()
		if err := matching.Persist(ctx, f.pool, f.articleID, []matching.Match{{CompanyID: f.post, Confidence: conf, Method: method}}); err != nil {
			t.Fatal(err)
		}
	}

	persist(0.55, "core_name")
	persist(0.90, "full_name")
	if got := current(t, f)[f.post]; got != "linked" {
		t.Fatalf("after upgrade: %q, want linked", got)
	}
	persist(0.55, "core_name")
	if got := current(t, f)[f.post]; got != "linked" {
		t.Fatalf("after weaker match: %q, want linked (no downgrade)", got)
	}
	if n := rowCount(t, f); n != 2 {
		t.Fatalf("%d rows, want 2 (pending, linked)", n)
	}
}

func TestPersistNeverResurrectsRejected(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := matching.Persist(ctx, f.pool, f.articleID, []matching.Match{{CompanyID: f.post, Confidence: 0.55, Method: "core_name"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by)
		VALUES ($1, $2, 0.55, 'admin', 'rejected', 'admin')`, f.articleID, f.post); err != nil {
		t.Fatal(err)
	}

	if err := matching.Persist(ctx, f.pool, f.articleID, []matching.Match{{CompanyID: f.post, Confidence: 1.0, Method: "issuer"}}); err != nil {
		t.Fatal(err)
	}
	if got := current(t, f)[f.post]; got != "rejected" {
		t.Fatalf("status = %q, want rejected", got)
	}
	if n := rowCount(t, f); n != 2 {
		t.Fatalf("%d rows, want 2", n)
	}
}

func TestPersistDropsBelowMinThreshold(t *testing.T) {
	f := setup(t)
	if err := matching.Persist(context.Background(), f.pool, f.articleID, []matching.Match{{CompanyID: f.post, Confidence: 0.2, Method: "core_name"}}); err != nil {
		t.Fatal(err)
	}
	if n := rowCount(t, f); n != 0 {
		t.Fatalf("%d rows, want 0", n)
	}
}
