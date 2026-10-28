package matching_test

import (
	"context"
	"os"
	"testing"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
	"github.com/rene404/aktien-news/backend/internal/matching"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
	"github.com/rene404/aktien-news/backend/seed"
)

// The real ad-hoc feed sample against the committed master data: only the
// Mercedes-Benz item concerns a seeded company; nothing else may link.
func TestRealFeedAgainstCommittedSeed(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	ix, err := matching.LoadIndex(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	mercedes, err := catalog.New(pool).CompanyByISIN(ctx, "DE0007100000")
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("../ingest/finanznachrichten/testdata/adhoc.xml")
	if err != nil {
		t.Fatal(err)
	}
	arts, err := finanznachrichten.Parse(b)
	if err != nil {
		t.Fatal(err)
	}

	for _, a := range arts {
		got := ix.Match(matching.Input{Title: a.Title, IssuerHint: a.IssuerHint, IssuerISIN: a.IssuerISIN, ISINIsIssuer: true})
		var linked []matching.Match
		for _, m := range got {
			if matching.StatusFor(m.Confidence) == "linked" {
				linked = append(linked, m)
			}
		}
		if a.IssuerISIN == "DE0007100000" {
			if len(linked) != 1 || linked[0].CompanyID != mercedes.ID || linked[0].Method != "issuer" {
				t.Errorf("mercedes item: linked = %+v", linked)
			}
			continue
		}
		if len(linked) != 0 {
			t.Errorf("%q linked unexpectedly: %+v", a.Title, linked)
		}
	}
}

// Regression: in FinanzNachrichten index feeds fn:isin is a loose tag, not
// the subject. Real titles from the first live run (2026-10-28).
func TestTaggedISINFromNewsFeedsNeedsCorroboration(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	ix, err := matching.LoadIndex(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		title, isin, wantStatus string
	}{
		{"Aktien Wien Schluss: ATX legt deutlich zu", "US02079K3059", "pending"},                           // Alphabet
		{"Aktien Wien Schluss: Moderate Gewinne am Ende einer schwachen Woche", "AT0000743059", "pending"}, // OMV
		{"UBS stuft DEUTSCHE BANK AG auf 'Buy'", "DE0005140008", "linked"},
		{"Siemens Energy Aktie: Vom KI-Gewinner zum KI-Opfer?", "DE000ENER6Y0", "linked"},
	}
	for _, tc := range cases {
		company, err := catalog.New(pool).CompanyByISIN(ctx, tc.isin)
		if err != nil {
			t.Fatal(err)
		}
		var status string
		for _, m := range ix.Match(matching.Input{Title: tc.title, IssuerISIN: tc.isin}) {
			if m.CompanyID == company.ID {
				status = matching.StatusFor(m.Confidence)
			}
		}
		if status != tc.wantStatus {
			t.Errorf("%q (%s): status %q, want %q", tc.title, company.Name, status, tc.wantStatus)
		}
	}
}
