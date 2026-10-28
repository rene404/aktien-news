package finanznachrichten_test

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
)

func parseFile(t *testing.T, name string) ([]ingest.RawArticle, error) {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return finanznachrichten.Parse(b)
}

func TestParseAdhocFixture(t *testing.T) {
	arts, err := parseFile(t, "adhoc.xml")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(arts) != 10 {
		t.Fatalf("got %d articles, want 10", len(arts))
	}

	first := arts[0]
	if !strings.HasPrefix(first.Title, "EQS-Adhoc: Befesa S.A.: Befesa raises FY 2026") {
		t.Errorf("title = %q", first.Title)
	}
	if want := time.Date(2026, 10, 28, 12, 3, 0, 0, time.UTC); !first.PublishedAt.Equal(want) {
		t.Errorf("published_at = %v, want %v", first.PublishedAt, want)
	}
	if first.IssuerISIN != "LU1704650164" {
		t.Errorf("isin = %q", first.IssuerISIN)
	}
	if first.IssuerHint != "Befesa S.A." {
		t.Errorf("issuer hint = %q", first.IssuerHint)
	}
	if !strings.HasPrefix(first.URL, "https://www.finanznachrichten.de/nachrichten-2026-10/") {
		t.Errorf("url = %q", first.URL)
	}
	// The feed has no <guid>; the link is the stable id.
	if first.ExternalID != first.URL {
		t.Errorf("external id = %q, want the link", first.ExternalID)
	}

	for _, a := range arts {
		if a.Title == "" || a.URL == "" || a.PublishedAt.IsZero() || a.IssuerISIN == "" {
			t.Errorf("incomplete article: %+v", a)
		}
	}
}

func TestParseDecodesEntitiesAndExtractsIssuer(t *testing.T) {
	arts, err := parseFile(t, "adhoc.xml")
	if err != nil {
		t.Fatal(err)
	}
	byISIN := map[string]ingest.RawArticle{}
	for _, a := range arts {
		byISIN[a.IssuerISIN] = a
	}

	cases := map[string]string{
		"DE0005706535": "EUROKAI GmbH & Co. KGaA", // &amp; decoded
		"CH1175448666": "Straumann Holding AG",    // "Adhoc:" prefix without EQS-
		"DE0007100000": "Mercedes-Benz Group AG",
	}
	for isin, want := range cases {
		if got := byISIN[isin].IssuerHint; got != want {
			t.Errorf("%s: issuer hint = %q, want %q", isin, got, want)
		}
	}
}

func TestIssuerHint(t *testing.T) {
	cases := map[string]string{
		"EQS-Adhoc: Turbon AG: Vorstand prüft Downlisting":           "Turbon AG",
		"EQS-News: PALFINGER AG: Neunmonatszahlen":                   "PALFINGER AG",
		"EQS-DD: Lenzing AG: Directors Dealings":                     "Lenzing AG",
		"Adhoc: Rieter AG: Rieter Investor Update 2026 - 9-Monats":   "Rieter AG",
		"DAX schließt freundlich":                                    "",
		"Analyse: Kursziel für SAP angehoben":                        "", // no second colon segment
		"EQS-Adhoc: IVU Traffic Technologies AG: Ad-hoc | IVU: Text": "IVU Traffic Technologies AG",
		"PTA-DD: BAWAG Group AG: Mitteilung über Eigengeschäfte":     "BAWAG Group AG",
		"PTA-Adhoc: VAS AG: Veröffentlichung des Prüfberichts":       "VAS AG",
		"PTA-News: Example SE: Hauptversammlung":                     "Example SE",
	}
	for title, want := range cases {
		if got := finanznachrichten.IssuerHint(title); got != want {
			t.Errorf("IssuerHint(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestParseEmptyFeed(t *testing.T) {
	arts, err := parseFile(t, "empty.xml")
	if err != nil {
		t.Fatalf("empty feed must not error: %v", err)
	}
	if len(arts) != 0 {
		t.Fatalf("got %d articles, want 0", len(arts))
	}
}

func TestParseBrokenFeed(t *testing.T) {
	if _, err := parseFile(t, "broken.xml"); err == nil {
		t.Fatal("broken feed: want error")
	}
}

func TestParseSkipsItemsWithoutLinkOrTitleAndAcceptsRFC1123Dates(t *testing.T) {
	feed := `<?xml version="1.0"?><rss version="2.0" xmlns:fn="http://www.finanznachrichten.de/service/rss"><channel>
	<item><title>EQS-Adhoc: A AG: ok</title><link>https://example.com/a</link><pubDate>Tue, 27 Oct 2026 09:03:00 +0100</pubDate><fn:isin>DE0007100000</fn:isin></item>
	<item><title>no link</title><pubDate>2026-10-27T09:03:00Z</pubDate></item>
	<item><link>https://example.com/no-title</link><pubDate>2026-10-27T09:03:00Z</pubDate></item>
	<item><title>bad date</title><link>https://example.com/c</link><pubDate>gestern</pubDate></item>
	</channel></rss>`
	arts, err := finanznachrichten.Parse([]byte(feed))
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 {
		t.Fatalf("got %d articles, want 1: %+v", len(arts), arts)
	}
	if want := time.Date(2026, 10, 27, 8, 3, 0, 0, time.UTC); !arts[0].PublishedAt.Equal(want) {
		t.Errorf("published_at = %v, want %v", arts[0].PublishedAt, want)
	}
}

// Legal invariant: feed descriptions (dpa-AFX snippets) are never carried.
func TestRawArticleHasNoTextField(t *testing.T) {
	typ := reflect.TypeOf(ingest.RawArticle{})
	for i := range typ.NumField() {
		name := strings.ToLower(typ.Field(i).Name)
		for _, banned := range []string{"desc", "body", "snippet", "summary", "content", "text"} {
			if strings.Contains(name, banned) {
				t.Errorf("RawArticle.%s looks like a text field (%q)", typ.Field(i).Name, banned)
			}
		}
	}
}
