package catalog_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/matching"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
	"github.com/rene404/aktien-news/backend/seed"
)

func TestParseSECTickers(t *testing.T) {
	b, err := os.ReadFile("testdata/sec/company_tickers_exchange.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.ParseSECTickers(b)
	if err != nil {
		t.Fatal(err)
	}
	// NYSE and Nasdaq only, one entry per CIK, class shares as BRK.B.
	want := []catalog.SECCompany{
		{CIK: "320193", Name: "Apple Inc.", Listings: []catalog.SECListing{{Ticker: "AAPL", MIC: "XNAS"}}},
		{CIK: "789019", Name: "MICROSOFT CORP", Listings: []catalog.SECListing{{Ticker: "MSFT", MIC: "XNAS"}}},
		{CIK: "1652044", Name: "Alphabet Inc.", Listings: []catalog.SECListing{{Ticker: "GOOGL", MIC: "XNAS"}, {Ticker: "GOOG", MIC: "XNAS"}}},
		{CIK: "1067983", Name: "BERKSHIRE HATHAWAY INC", Listings: []catalog.SECListing{{Ticker: "BRK.B", MIC: "XNYS"}, {Ticker: "BRK.A", MIC: "XNYS"}}},
		{CIK: "1090872", Name: "Agilent Technologies, Inc.", Listings: []catalog.SECListing{{Ticker: "A", MIC: "XNYS"}}},
		{CIK: "1097864", Name: "ON SEMICONDUCTOR CORP", Listings: []catalog.SECListing{{Ticker: "ON", MIC: "XNAS"}}},
		{CIK: "749251", Name: "GARTNER INC", Listings: []catalog.SECListing{{Ticker: "IT", MIC: "XNYS"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseSECTickersRejectsUnexpectedShape(t *testing.T) {
	for _, in := range []string{`{`, `{"fields":["cik","name"],"data":[]}`, `{"fields":["cik","name","ticker","exchange"],"data":[["x","A","B","NYSE"]]}`} {
		if _, err := catalog.ParseSECTickers([]byte(in)); err == nil {
			t.Errorf("ParseSECTickers(%s): want error", in)
		}
	}
}

type companyRow struct {
	slug, name, cik string
	curated         bool
	tickers         string
	aliases         int
}

func companies(t *testing.T, pool *pgxpool.Pool) map[string]companyRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT c.slug, c.name, coalesce(c.cik, ''), c.curated,
		       coalesce((SELECT string_agg(l.mic || ':' || l.ticker, ' ' ORDER BY l.mic, l.ticker) FROM listings l WHERE l.company_id = c.id), ''),
		       (SELECT count(*) FROM company_aliases a WHERE a.company_id = c.id)
		FROM companies c`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]companyRow{}
	for rows.Next() {
		var r companyRow
		if err := rows.Scan(&r.slug, &r.name, &r.cik, &r.curated, &r.tickers, &r.aliases); err != nil {
			t.Fatal(err)
		}
		out[r.slug] = r
	}
	return out
}

func TestImportSEC(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	// A curated company the seed knows without CIK: matched by its ticker.
	if _, err := pool.Exec(ctx, `UPDATE companies SET cik = NULL WHERE slug = 'microsoft'`); err != nil {
		t.Fatal(err)
	}
	before := companies(t, pool)
	b, err := os.ReadFile("testdata/sec/company_tickers_exchange.json")
	if err != nil {
		t.Fatal(err)
	}
	secs, err := catalog.ParseSECTickers(b)
	if err != nil {
		t.Fatal(err)
	}

	for run := 1; run <= 2; run++ { // idempotent
		if err := catalog.ImportSEC(ctx, pool, secs); err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		got := companies(t, pool)
		if len(got) != len(before)+3 {
			t.Fatalf("run %d: %d companies, want %d", run, len(got), len(before)+3)
		}
		// Curated companies keep names, aliases and listings; a missing CIK is filled in.
		for _, slug := range []string{"apple", "alphabet", "berkshire-hathaway", "microsoft"} {
			want := before[slug]
			if slug == "microsoft" {
				want.cik = "789019"
			}
			if got[slug] != want {
				t.Errorf("run %d: %s = %+v, want %+v", run, slug, got[slug], want)
			}
		}
		for slug, want := range map[string]companyRow{
			"sec-1090872": {slug: "sec-1090872", name: "Agilent Technologies, Inc.", cik: "1090872", tickers: "XNYS:A"},
			"sec-1097864": {slug: "sec-1097864", name: "ON SEMICONDUCTOR CORP", cik: "1097864", tickers: "XNAS:ON"},
			"sec-749251":  {slug: "sec-749251", name: "GARTNER INC", cik: "749251", tickers: "XNYS:IT"},
		} {
			if got[slug] != want {
				t.Errorf("run %d: %s = %+v, want %+v", run, slug, got[slug], want)
			}
		}
	}

	// Re-seeding the curated master data does not wipe CIKs it does not list.
	if _, err := pool.Exec(ctx, `UPDATE companies SET cik = '789019' WHERE slug = 'microsoft'`); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Seed(ctx, pool, os.DirFS("testdata/sec/seed_without_cik")); err != nil {
		t.Fatal(err)
	}
	if got := companies(t, pool)["microsoft"].cik; got != "789019" {
		t.Errorf("cik after re-seed = %q, want 789019", got)
	}
}

func TestImportedCompaniesMatchOnlyOnStrongSignals(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/sec/company_tickers_exchange.json")
	if err != nil {
		t.Fatal(err)
	}
	secs, err := catalog.ParseSECTickers(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.ImportSEC(ctx, pool, secs); err != nil {
		t.Fatal(err)
	}
	ix, err := matching.LoadIndex(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	var agilent int64
	if err := pool.QueryRow(ctx, `SELECT id FROM companies WHERE cik = '1090872'`).Scan(&agilent); err != nil {
		t.Fatal(err)
	}

	if got := ix.Match(matching.Input{Title: "Plan A: ON track, IT spending up"}); len(got) != 0 {
		t.Errorf("short tickers matched: %+v", got)
	}
	got := ix.Match(matching.Input{Title: "Agilent Technologies, Inc. Reports Third-Quarter Results"})
	if len(got) != 1 || got[0].CompanyID != agilent || matching.StatusFor(got[0].Confidence) != "linked" {
		t.Errorf("full name: %+v, want Agilent linked", got)
	}
	// Curated companies still match on their ticker.
	if got := ix.Match(matching.Input{Title: "MSFT hits record"}); len(got) != 1 || got[0].Method != "ticker" {
		t.Errorf("curated ticker: %+v", got)
	}
}

func countCompanies(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM companies`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// European issuers with a US listing (ADR) are in the seed without CIK and
// with home-exchange tickers only; the legal name ties them to the SEC row.
func TestImportSECMatchesCuratedByLegalName(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	before := countCompanies(t, pool)
	secs := []catalog.SECCompany{
		{CIK: "1000184", Name: "SAP SE", Listings: []catalog.SECListing{{Ticker: "SAP", MIC: "XNYS"}}},
		{CIK: "1159508", Name: "DEUTSCHE BANK AKTIENGESELLSCHAFT", Listings: []catalog.SECListing{{Ticker: "DB", MIC: "XNYS"}}},
	}
	if err := catalog.ImportSEC(ctx, pool, secs); err != nil {
		t.Fatal(err)
	}
	got := companies(t, pool)
	if got["sap"].cik != "1000184" || got["deutsche-bank"].cik != "1159508" {
		t.Errorf("sap = %+v, deutsche-bank = %+v", got["sap"], got["deutsche-bank"])
	}
	if n := countCompanies(t, pool); n != before {
		t.Errorf("%d companies, want %d (no duplicates)", n, before)
	}
}

// An earlier import created an uncurated duplicate; the next import folds it
// into the curated company.
func TestImportSECRemovesEarlierDuplicate(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	before := countCompanies(t, pool)
	if _, err := pool.Exec(ctx, `
		WITH c AS (INSERT INTO companies (slug, name, legal_name, country, cik, curated)
		           VALUES ('sec-1000184', 'SAP SE', 'SAP SE', 'US', '1000184', false) RETURNING id)
		INSERT INTO listings (company_id, ticker, mic) SELECT id, 'SAP', 'XNYS' FROM c`); err != nil {
		t.Fatal(err)
	}

	err := catalog.ImportSEC(ctx, pool, []catalog.SECCompany{
		{CIK: "1000184", Name: "SAP SE", Listings: []catalog.SECListing{{Ticker: "SAP", MIC: "XNYS"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := companies(t, pool)
	if _, dup := got["sec-1000184"]; dup || got["sap"].cik != "1000184" {
		t.Errorf("duplicate kept = %v, sap = %+v", dup, got["sap"])
	}
	if n := countCompanies(t, pool); n != before {
		t.Errorf("%d companies, want %d", n, before)
	}
}

// Adding a curated company whose CIK an import already gave to an uncurated
// row must not break the seed: the curated row takes over.
func TestSeedTakesOverUncuratedCompanyWithSameCIK(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("testdata/valid")); err != nil {
		t.Fatal(err)
	}
	if err := catalog.ImportSEC(ctx, pool, []catalog.SECCompany{
		{CIK: "1999999", Name: "NEWCO INC", Listings: []catalog.SECListing{{Ticker: "NEWC", MIC: "XNAS"}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Seed(ctx, pool, os.DirFS("testdata/sec/seed_new_curated")); err != nil {
		t.Fatalf("seed after import: %v", err)
	}
	got := companies(t, pool)
	if _, dup := got["sec-1999999"]; dup {
		t.Error("uncurated sec-1999999 still exists")
	}
	if want := (companyRow{slug: "newco", name: "NewCo", cik: "1999999", curated: true, tickers: "XNAS:NEWC"}); got["newco"] != want {
		t.Errorf("newco = %+v, want %+v", got["newco"], want)
	}
}
