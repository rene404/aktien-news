package catalog_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

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

func TestSeedLinksIRSourceToIssuer(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("testdata/ir_source")); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, `
		SELECT s.key, coalesce(c.slug, '') FROM sources s
		LEFT JOIN companies c ON c.id = s.issuer_company_id ORDER BY s.key`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	got := map[string]string{}
	for rows.Next() {
		var key, slug string
		if err := rows.Scan(&key, &slug); err != nil {
			t.Fatal(err)
		}
		got[key] = slug
	}
	want := map[string]string{"fn_adhoc": "", "post_ir": "oesterreichische-post"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("source issuers = %v, want %v", got, want)
	}
}

func TestSeedRejectsUnknownIssuerSlug(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)

	err := catalog.Seed(ctx, pool, os.DirFS("testdata/unknown_issuer"))
	if err == nil || !strings.Contains(err.Error(), `unknown issuer_slug "acme"`) {
		t.Fatalf("err = %v, want an error naming the unknown issuer_slug", err)
	}
	for table, n := range countRows(t, pool) {
		if n != 0 {
			t.Errorf("%s has %d rows after a failed seed, want 0", table, n)
		}
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

// fixture is testdata/valid with some files replaced.
func fixture(t *testing.T, replace map[string]string) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{}
	for _, name := range []string{"sources.csv", "companies.csv", "listings.csv", "index_memberships.csv", "company_aliases.csv"} {
		b, err := os.ReadFile("testdata/valid/" + name)
		if err != nil {
			t.Fatal(err)
		}
		fsys[name] = &fstest.MapFile{Data: b}
	}
	for name, data := range replace {
		fsys[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return fsys
}

// docs/plan-eu-sources.md, task 1: France, Spain and the Netherlands with their exchanges.
func TestSeedAcceptsEUMarketsAndExchanges(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	err := catalog.Seed(ctx, pool, fixture(t, map[string]string{
		"sources.csv": `key,name,kind,market,url,parser,weight,fetch_interval_minutes,text_policy,terms_url,issuer_slug
amf,AMF,adhoc,FR,https://example.com/amf,dila,1.00,60,none,,
cnmv,CNMV,adhoc,ES,https://example.com/cnmv,cnmv,1.00,60,none,,
afm,AFM,filing,NL,https://example.com/afm,afm,1.00,60,none,,
`,
		"companies.csv": `slug,name,legal_name,country,lei,cik
totalenergies,TotalEnergies,TotalEnergies SE,FR,,
inditex,Inditex,"Industria de Diseño Textil, S.A.",ES,,
asml,ASML,ASML Holding N.V.,NL,,
`,
		"listings.csv": `company_slug,isin,ticker,mic,currency
totalenergies,FR0000120271,TTE,XPAR,EUR
inditex,ES0148396007,ITX,XMAD,EUR
asml,NL0010273215,ASML,XAMS,EUR
`,
		"index_memberships.csv": "mic,ticker,index_key,valid_from\n",
		"company_aliases.csv":   "company_slug,alias,kind\n",
	}))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var markets string
	if err := pool.QueryRow(ctx, `SELECT string_agg(market, ',' ORDER BY market) FROM sources`).Scan(&markets); err != nil {
		t.Fatal(err)
	}
	if markets != "ES,FR,NL" {
		t.Errorf("source markets = %s, want ES,FR,NL", markets)
	}
}

func TestSeedStillRejectsUnknownMarketsAndExchanges(t *testing.T) {
	for _, tc := range []struct {
		name, file, data, want string
	}{
		{"market", "sources.csv", `key,name,kind,market,url,parser,weight,fetch_interval_minutes,text_policy,terms_url,issuer_slug
lse,LSE,adhoc,UK,https://example.com/lse,lse,1.00,60,none,,
`, `unknown market "UK"`},
		{"exchange", "listings.csv", `company_slug,isin,ticker,mic,currency
apple,US0378331005,AAPL,XLON,USD
`, `unknown mic "XLON"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := testdb.New(t)
			err := catalog.Seed(context.Background(), pool, fixture(t, map[string]string{tc.file: tc.data}))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %s", err, tc.want)
			}
		})
	}
}

// ISO 17442: 18 alphanumerics plus two check digits, mod 97 = 1 (as IBAN).
func TestValidLEI(t *testing.T) {
	cases := map[string]bool{
		"529900S21EQ1BO4ESM68": true,  // TotalEnergies
		"95980020140005693107": true,  // MAPFRE (digits only)
		"724500Y6DUVHQD6OXN27": true,  // ASML
		"529900S21EQ1BO4ESM69": false, // wrong check digits
		"529900S21EQ1BO4ESM6":  false, // too short
		"529900s21eq1bo4esm68": false, // lowercase
		"529900S21EQ1BO4ESM6Ö": false,
		"":                     false,
	}
	for lei, want := range cases {
		if got := catalog.ValidLEI(lei); got != want {
			t.Errorf("ValidLEI(%q) = %v, want %v", lei, got, want)
		}
	}
}

func TestSeedRejectsInvalidLEI(t *testing.T) {
	pool := testdb.New(t)
	err := catalog.Seed(context.Background(), pool, fixture(t, map[string]string{
		"companies.csv": `slug,name,legal_name,country,lei,cik
erste-group,Erste Group,Erste Group Bank AG,AT,PQOH26KWDF7CG10L6793,
oesterreichische-post,Österreichische Post,Österreichische Post AG,AT,,
apple,Apple,"Apple Inc.",US,,320193
`,
	}))
	if err == nil || !strings.Contains(err.Error(), "invalid LEI PQOH26KWDF7CG10L6793") {
		t.Fatalf("err = %v, want an error naming the bad LEI", err)
	}
}

// docs/plan-eu-sources.md, task 3: CAC 40, IBEX 35 and AEX are complete,
// every member has a listing on its exchange and an issuer LEI.
func TestCommittedSeedHasEUIndices(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		index string
		n     int
		mics  string
	}{
		{"CAC40", 40, "XAMS,XPAR"}, // ArcelorMittal counts with its Amsterdam line
		{"IBEX35", 35, "XMAD"},
		{"AEX", 30, "XAMS"},
		// Wikipedia and Nasdaq's review notices as of 2026-11-09. Nordea is in
		// all three, with one line per exchange.
		{"OMXS30", 30, "XSTO"},
		{"OMXH25", 25, "XHEL"},
		{"OMXC25", 25, "XCSE"},
	} {
		var n, withoutLEI int
		var mics string
		if err := pool.QueryRow(ctx, `
			SELECT count(*), count(*) FILTER (WHERE c.lei IS NULL), coalesce(string_agg(DISTINCT l.mic, ',' ORDER BY l.mic), '')
			FROM index_memberships m JOIN listings l ON l.id = m.listing_id JOIN companies c ON c.id = l.company_id
			WHERE m.index_key = $1 AND m.valid_to IS NULL`, tc.index).Scan(&n, &withoutLEI, &mics); err != nil {
			t.Fatal(err)
		}
		if n != tc.n || withoutLEI != 0 || mics != tc.mics {
			t.Errorf("%s: %d members (want %d), %d without LEI, exchanges %s (want %s)", tc.index, n, tc.n, withoutLEI, mics, tc.mics)
		}
	}
}
