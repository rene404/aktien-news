package store_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/store"
	"github.com/rene404/aktien-news/backend/internal/store/db"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
)

var wantTables = []string{
	"article_entities", "article_links", "article_texts", "articles", "classifications", "clusters",
	"companies", "company_aliases", "external_sentiment", "index_memberships",
	"ingest_runs", "listings", "sources",
}

func publicTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name <> 'goose_db_version'
		ORDER BY table_name`)
	if err != nil {
		t.Fatal(err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func TestMigrationsRoundTrip(t *testing.T) {
	ctx := context.Background()
	dsn := testdb.NewEmpty(t)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if err := store.MigrateUp(ctx, dsn); err != nil {
		t.Fatalf("up: %v", err)
	}
	if got := publicTables(t, ctx, pool); !slices.Equal(got, wantTables) {
		t.Fatalf("after up: tables = %v, want %v", got, wantTables)
	}

	if err := store.MigrateDown(ctx, dsn); err != nil {
		t.Fatalf("down: %v", err)
	}
	if got := publicTables(t, ctx, pool); len(got) != 0 {
		t.Fatalf("after down: leftover tables %v", got)
	}

	if err := store.MigrateUp(ctx, dsn); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := publicTables(t, ctx, pool); !slices.Equal(got, wantTables) {
		t.Fatalf("after up again: tables = %v", got)
	}
}

func seedSource(t *testing.T, ctx context.Context, q *db.Queries) db.Source {
	t.Helper()
	src, err := q.UpsertSource(ctx, db.UpsertSourceParams{
		Key:    "fn_adhoc",
		Name:   "FinanzNachrichten Ad-hoc",
		Kind:   db.SourceKindAdhoc,
		Market: "DE",
		Url:    "https://www.finanznachrichten.de/rss-aktien-adhoc/",
	})
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func newArticle(sourceID int64, externalID, url string) db.InsertArticleParams {
	return db.InsertArticleParams{
		SourceID:     sourceID,
		ExternalID:   externalID,
		Url:          url,
		CanonicalUrl: url,
		Title:        "EQS-Adhoc: Beispiel AG hebt Prognose an",
		PublishedAt:  time.Date(2026, 10, 28, 7, 30, 0, 0, time.UTC),
		ContentHash:  []byte{1, 2, 3},
	}
}

func TestInsertArticleSetsIngestedAtAndDedupes(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	q := db.New(pool)
	src := seedSource(t, ctx, q)

	before := time.Now().Add(-time.Minute)
	a, err := q.InsertArticle(ctx, newArticle(src.ID, "guid-1", "https://example.com/a"))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if a.IngestedAt.Before(before) {
		t.Errorf("ingested_at %v not set by the database", a.IngestedAt)
	}

	// Same (source_id, external_id) again: no new row, no error.
	_, err = q.InsertArticle(ctx, newArticle(src.ID, "guid-1", "https://example.com/other"))
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("duplicate external_id: err = %v, want pgx.ErrNoRows", err)
	}
	// Same canonical URL from another item: also deduplicated.
	_, err = q.InsertArticle(ctx, newArticle(src.ID, "guid-2", "https://example.com/a"))
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("duplicate canonical_url: err = %v, want pgx.ErrNoRows", err)
	}
}

func TestIngestedAtIsImmutable(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	q := db.New(pool)
	a, err := q.InsertArticle(ctx, newArticle(seedSource(t, ctx, q).ID, "guid-1", "https://example.com/a"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `UPDATE articles SET ingested_at = now() - interval '1 day' WHERE id = $1`, a.ID)
	assertPgError(t, err, "P0001")
}

func TestAppendOnlyTablesRejectUpdates(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	q := db.New(pool)
	a, err := q.InsertArticle(ctx, newArticle(seedSource(t, ctx, q).ID, "guid-1", "https://example.com/a"))
	if err != nil {
		t.Fatal(err)
	}
	var companyID int64
	if err := pool.QueryRow(ctx, `INSERT INTO companies (slug, name, country) VALUES ('beispiel', 'Beispiel AG', 'DE') RETURNING id`).Scan(&companyID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by)
		VALUES ($1, $2, 0.98, 'isin', 'linked', 'engine')`, a.ID, companyID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO classifications (article_id, company_id, event_type, sentiment, confidence, novelty, model, prompt_version)
		VALUES ($1, $2, 'guidance', 0.6, 0.7, 1, 'fake', 'v1')`, a.ID, companyID); err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `UPDATE article_entities SET status = 'rejected'`)
	assertPgError(t, err, "P0001")
	_, err = pool.Exec(ctx, `UPDATE classifications SET sentiment = -1`)
	assertPgError(t, err, "P0001")
}

func TestSentimentRangeIsChecked(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	q := db.New(pool)
	a, err := q.InsertArticle(ctx, newArticle(seedSource(t, ctx, q).ID, "guid-1", "https://example.com/a"))
	if err != nil {
		t.Fatal(err)
	}
	var companyID int64
	if err := pool.QueryRow(ctx, `INSERT INTO companies (slug, name, country) VALUES ('beispiel', 'Beispiel AG', 'DE') RETURNING id`).Scan(&companyID); err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO classifications (article_id, company_id, event_type, sentiment, confidence, novelty, model, prompt_version)
		VALUES ($1, $2, 'other', 1.5, 0.5, 1, 'fake', 'v1')`, a.ID, companyID)
	assertPgError(t, err, "23514") // check_violation
}

func assertPgError(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("err = %v, want postgres error %s", err, code)
	}
	if pgErr.Code != code {
		t.Fatalf("postgres error code = %s (%s), want %s", pgErr.Code, pgErr.Message, code)
	}
}

// article_links is a trigger-maintained projection: one row per
// (article, company) holding the newest decision from article_entities.
func TestArticleLinksTracksNewestDecision(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	q := db.New(pool)
	a, err := q.InsertArticle(ctx, newArticle(seedSource(t, ctx, q).ID, "guid-1", "https://example.com/a"))
	if err != nil {
		t.Fatal(err)
	}
	var companyID int64
	if err := pool.QueryRow(ctx, `INSERT INTO companies (slug, name, country) VALUES ('beispiel', 'Beispiel AG', 'DE') RETURNING id`).Scan(&companyID); err != nil {
		t.Fatal(err)
	}
	decide := func(status string, at time.Time) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_at, decided_by)
			VALUES ($1, $2, 0.9, 'ticker', $3, $4, 'engine')`, a.ID, companyID, status, at); err != nil {
			t.Fatal(err)
		}
	}
	current := func() string {
		t.Helper()
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM article_links WHERE article_id = $1 AND company_id = $2`, a.ID, companyID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		return status
	}
	t0 := time.Date(2026, 10, 28, 8, 0, 0, 0, time.UTC)

	decide("pending", t0)
	if got := current(); got != "pending" {
		t.Fatalf("after first decision: %s", got)
	}
	decide("linked", t0.Add(time.Minute))
	if got := current(); got != "linked" {
		t.Fatalf("after upgrade: %s", got)
	}
	decide("rejected", t0.Add(30*time.Second)) // older than the current row
	if got := current(); got != "linked" {
		t.Fatalf("an older decision overwrote the newer one: %s", got)
	}
	decide("rejected", t0.Add(time.Minute)) // same time, later id wins
	if got := current(); got != "rejected" {
		t.Fatalf("tie on decided_at must go to the later row: %s", got)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM article_links`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("article_links rows = %d, want 1", n)
	}
}
