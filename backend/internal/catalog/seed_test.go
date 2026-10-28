package catalog_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
	"github.com/rene404/aktien-news/backend/seed"
)

func countRows(t *testing.T, pool *pgxpool.Pool) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, table := range []string{"sources", "companies", "listings", "index_memberships", "company_aliases"} {
		var n int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		counts[table] = n
	}
	return counts
}

func TestSeedIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	data := os.DirFS("testdata/valid")

	if err := catalog.Seed(ctx, pool, data); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	first := countRows(t, pool)
	want := map[string]int{"sources": 1, "companies": 3, "listings": 3, "index_memberships": 3, "company_aliases": 2}
	for table, n := range want {
		if first[table] != n {
			t.Errorf("%s: %d rows, want %d", table, first[table], n)
		}
	}

	if err := catalog.Seed(ctx, pool, data); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	if second := countRows(t, pool); !mapsEqual(first, second) {
		t.Fatalf("second seed changed row counts: %v -> %v", first, second)
	}
}

func TestLookups(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("testdata/valid")); err != nil {
		t.Fatal(err)
	}
	cat := catalog.New(pool)

	byISIN, err := cat.CompanyByISIN(ctx, "AT0000APOST4")
	if err != nil {
		t.Fatalf("by isin: %v", err)
	}
	if byISIN.Slug != "oesterreichische-post" {
		t.Errorf("by isin: slug = %q", byISIN.Slug)
	}

	byTicker, err := cat.CompanyByTicker(ctx, "XNAS", "AAPL")
	if err != nil {
		t.Fatalf("by ticker: %v", err)
	}
	if byTicker.Slug != "apple" || byTicker.Cik == nil || *byTicker.Cik != "320193" {
		t.Errorf("by ticker: got %+v", byTicker)
	}

	if _, err := cat.CompanyByISIN(ctx, "DE0007164600"); err != catalog.ErrNotFound {
		t.Errorf("unknown isin: err = %v, want ErrNotFound", err)
	}
}

func TestSeedRejectsInvalidISINAndWritesNothing(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)

	err := catalog.Seed(ctx, pool, os.DirFS("testdata/bad_isin"))
	if err == nil || !strings.Contains(err.Error(), "AT0000652012") {
		t.Fatalf("err = %v, want an error naming the bad ISIN", err)
	}
	for table, n := range countRows(t, pool) {
		if n != 0 {
			t.Errorf("%s has %d rows after a failed seed, want 0", table, n)
		}
	}
}

func TestValidISIN(t *testing.T) {
	cases := map[string]bool{
		"US0378331005": true,  // Apple
		"AT0000652011": true,  // Erste Group
		"AT0000APOST4": true,  // Österreichische Post (letters in NSIN)
		"DE0007164600": true,  // SAP
		"AT0000652012": false, // wrong check digit
		"AT000065201":  false, // too short
		"at0000652011": false, // lowercase country
		"":             false,
	}
	for isin, want := range cases {
		if got := catalog.ValidISIN(isin); got != want {
			t.Errorf("ValidISIN(%q) = %v, want %v", isin, got, want)
		}
	}
}

// The committed master data must always load cleanly.
func TestCommittedSeedFilesLoad(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatalf("seed/: %v", err)
	}
}

func mapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
