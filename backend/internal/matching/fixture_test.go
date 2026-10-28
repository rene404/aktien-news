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
		got := ix.Match(matching.Input{Title: a.Title, IssuerHint: a.IssuerHint, IssuerISIN: a.IssuerISIN})
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
