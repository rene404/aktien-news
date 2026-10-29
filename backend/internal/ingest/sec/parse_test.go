package sec_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/sec"
)

func parseFile(t *testing.T, name string) ([]ingest.RawArticle, error) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return sec.Parse(b)
}

func TestParse8K(t *testing.T) {
	arts, err := parseFile(t, "current_8k.xml")
	if err != nil {
		t.Fatal(err)
	}
	edt := time.FixedZone("EDT", -4*3600)
	want := []ingest.RawArticle{
		{
			ExternalID:  "0000320193-26-000080",
			URL:         "https://www.sec.gov/Archives/edgar/data/320193/000032019326000080/0000320193-26-000080-index.htm",
			Title:       "8-K: APPLE INC – Item 2.02 Results of Operations and Financial Condition",
			PublishedAt: time.Date(2026, 10, 29, 16, 30, 41, 0, edt).UTC(),
			IssuerCIK:   "320193", // leading zeros dropped, as in companies.cik
		},
		{
			ExternalID:  "0001090872-26-000031",
			URL:         "https://www.sec.gov/Archives/edgar/data/1090872/000109087226000031/0001090872-26-000031-index.htm",
			Title:       "8-K: Agilent Technologies, Inc. – Item 5.02 Departure of Directors or Certain Officers; Election of Directors; Appointment of Certain Officers; Compensatory Arrangements of Certain Officers · Item 7.01 Regulation FD Disclosure",
			PublishedAt: time.Date(2026, 10, 29, 16, 5, 2, 0, edt).UTC(),
			IssuerCIK:   "1090872",
		},
		{
			ExternalID:  "0001999999-26-000002",
			URL:         "https://www.sec.gov/Archives/edgar/data/1999999/000199999926000002/0001999999-26-000002-index.htm",
			Title:       "8-K/A: SMALLCAP WIDGETS CORP – Item 8.01 Other Events",
			PublishedAt: time.Date(2026, 10, 29, 15, 58, 0, 0, edt).UTC(),
			IssuerCIK:   "1999999",
		},
	}
	// The (Subject) row of a third-party filing and the entry without a CIK are skipped.
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseEmptyFeed(t *testing.T) {
	arts, err := parseFile(t, "empty.xml")
	if err != nil || len(arts) != 0 {
		t.Fatalf("arts = %v, err = %v", arts, err)
	}
}

func TestParseBrokenFeedIsError(t *testing.T) {
	if _, err := parseFile(t, "broken.xml"); err == nil {
		t.Fatal("want error")
	}
}

func TestParseDecodesLatin1(t *testing.T) {
	arts, err := parseFile(t, "latin1.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 || arts[0].Title != "8-K: SOCIÉTÉ EXAMPLE SA – Item 8.01 Other Events" || arts[0].IssuerCIK != "123456" {
		t.Errorf("got %+v", arts)
	}
}
