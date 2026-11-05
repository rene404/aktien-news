package pressrss_test

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/pressrss"
)

func parseFile(t *testing.T, name string) ([]ingest.RawArticle, error) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return pressrss.Parse(b)
}

func TestParseOTS(t *testing.T) {
	arts, err := parseFile(t, "ots_finanzen.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 15 {
		t.Fatalf("got %d articles, want 15", len(arts))
	}
	first := arts[0]
	want := ingest.RawArticle{
		ExternalID:  "https://www.ots.at/presseaussendung/EQS_20261029_EQS0006/eqs-news-lenzing-group-provides-update-on-shareholder-structure-following-completion-of-capital-increase",
		URL:         "https://www.ots.at/presseaussendung/EQS_20261029_EQS0006/eqs-news-lenzing-group-provides-update-on-shareholder-structure-following-completion-of-capital-increase",
		Title:       "EQS-News: Lenzing Group provides update on shareholder structure following completion of capital increase",
		PublishedAt: time.Date(2026, 10, 29, 7, 0, 37, 0, time.UTC), // 08:00:37 +0100
		IssuerHint:  "Lenzing AG",
	}
	if !reflect.DeepEqual(first, want) {
		t.Errorf("first =\n%+v\nwant\n%+v", first, want)
	}
	for _, a := range arts {
		if a.Title == "" || a.URL == "" || a.PublishedAt.IsZero() || a.IssuerHint == "" {
			t.Errorf("incomplete article: %+v", a)
		}
	}
}

func TestParsePRNewswire(t *testing.T) {
	arts, err := parseFile(t, "prn.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 20 {
		t.Fatalf("got %d articles, want 20", len(arts))
	}
	if got := arts[0].IssuerHint; got != "Contemporary Amperex Technology Co., Limited (CATL)" {
		t.Errorf("issuer hint = %q", got)
	}
	if want := time.Date(2026, 10, 29, 7, 40, 0, 0, time.UTC); !arts[0].PublishedAt.Equal(want) {
		t.Errorf("published_at = %v, want %v", arts[0].PublishedAt, want)
	}
	if got := arts[1].Title; !strings.Contains(got, "R&D Investment") {
		t.Errorf("title not entity-decoded: %q", got)
	}
	for _, a := range arts {
		if a.IssuerHint == "" || a.IssuerISIN != "" {
			t.Errorf("article %q: issuer hint %q, isin %q", a.Title, a.IssuerHint, a.IssuerISIN)
		}
	}
}

// Deutsche Telekom's IR feed (real response of 2026-10-29). It names no
// sender; the source's issuer_slug links every item to the company.
func TestParseTelekomIR(t *testing.T) {
	arts, err := parseFile(t, "telekom_ir.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 10 {
		t.Fatalf("got %d articles, want 10", len(arts))
	}
	link := "https://www.telekom.com/en/newsroom/latest-updates/media-information/2026/10/deutsche-telekom-boosts-growth-efficiency-and-quality-through-t"
	want := ingest.RawArticle{
		ExternalID: link, URL: link,
		Title:       "Deutsche Telekom boosts growth, efficiency and quality through the use of AI",
		PublishedAt: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
	}
	if !reflect.DeepEqual(arts[0], want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts[0], want)
	}
}

func TestParseWithoutIssuerAndIncompleteItems(t *testing.T) {
	arts, err := parseFile(t, "no_issuer.xml")
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{ExternalID: "https://example.com/a", URL: "https://example.com/a", Title: "Ohne Aussender",
			PublishedAt: time.Date(2026, 10, 29, 7, 0, 37, 0, time.UTC)},
		{ExternalID: "b-1", URL: "https://example.com/b", Title: "Empty contributor",
			PublishedAt: time.Date(2026, 10, 29, 7, 40, 0, 0, time.UTC)},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseEmptyFeed(t *testing.T) {
	arts, err := parseFile(t, "empty.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 0 {
		t.Errorf("got %d articles, want 0", len(arts))
	}
}

func TestParseBrokenFeedIsError(t *testing.T) {
	if _, err := parseFile(t, "broken.xml"); err == nil {
		t.Fatal("want error for truncated feed")
	}
}
