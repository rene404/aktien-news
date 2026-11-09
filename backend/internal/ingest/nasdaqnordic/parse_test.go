package nasdaqnordic_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/nasdaqnordic"
)

const view = "https://view.news.eu.nasdaq.com/view?id="

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func at(s string) time.Time {
	ts, err := time.Parse(time.DateTime, s)
	if err != nil {
		panic(err)
	}
	return ts.UTC()
}

// Real announcements of 2026-11-06/09 (the feed is queried with timeZone=UTC).
// Issuers publish the English and the local version as two announcements;
// the English one is kept. Iceland is no target market.
func TestParseAdhoc(t *testing.T) {
	arts, err := nasdaqnordic.ParseAdhoc(read(t, "main.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{
			ExternalID:  "1475904",
			URL:         view + "b92e74a35d95f847dea49123789f18dd2&lang=en&src=listed",
			Title:       "Alm. Brand A/S - Alm. Brand A/S announces invitation to holders of its outstanding DKK 400 million Perpetual Restricted Tier 1 Capital Notes to tender notes for repurchase",
			PublishedAt: at("2026-11-09 09:36:36"),
			IssuerHint:  "Alm. Brand A/S",
		},
		{
			// Only in Danish; the title does not name the issuer.
			ExternalID:  "1475831",
			URL:         view + "b3e15789bb876e2cbeabe50da7c64935f&lang=da&src=listed",
			Title:       "Nordea Kredit Realkreditaktieselskab: Nr. 107, 2026 - Ekstraordinære indfrielser (CK93) pr. 6. november 2026",
			PublishedAt: at("2026-11-09 07:36:55"),
			IssuerHint:  "Nordea Kredit Realkreditaktieselskab",
		},
		{
			// The Swedish twin was filed a second earlier.
			ExternalID:  "1475809",
			URL:         view + "b48f3071365a45dfb6c1716b2db211fd8&lang=en&src=listed",
			Title:       "Svea Bank announces a public cash offer to the shareholders in Qliro",
			PublishedAt: at("2026-11-09 07:00:11"),
			IssuerHint:  "Svea Bank AB",
		},
		{
			ExternalID:  "1475515",
			URL:         view + "bf3f2699c0ba4181f9901da5d622500b4&lang=en&src=listed",
			Title:       "Inside information, profit warning: Raisio lowers its guidance for 2026 and provides preliminary information on the financial development of the third quarter",
			PublishedAt: at("2026-11-06 07:30:00"),
			IssuerHint:  "Raisio",
		},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

// Regulatory announcements other than inside information. Managers'
// transactions name the issuer only, never the person
// (docs/plan-eu-sources.md, decision 2).
func TestParseFilings(t *testing.T) {
	arts, err := nasdaqnordic.ParseFilings(read(t, "main.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range arts {
		got = append(got, a.ExternalID+" | "+a.Title+" | "+a.PublishedAt.Format(time.RFC3339))
	}
	want := []string{
		"1475881 | Buyback of Class B shares in Essity during week 45, 2026 | 2026-11-09T09:03:16Z",
		"1475855 | NORDEN: Eigengeschäft einer Führungskraft | 2026-11-09T08:22:57Z",
		"1475588 | Huhtamäki Oyj: Eigengeschäft einer Führungskraft | 2026-11-06T10:00:00Z",
		// "Ericsson, Telefonab. L M": the title names it.
		"1475827 | Share buybacks in Ericsson during the period November 2 - November 6, 2026 | 2026-11-09T07:30:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	for _, a := range arts {
		if strings.Contains(a.Title, "Person") {
			t.Errorf("title names a person: %q", a.Title)
		}
	}
}

// Investor news: the issuers' own, non-regulatory press releases.
func TestParseNews(t *testing.T) {
	arts, err := nasdaqnordic.ParseNews(read(t, "main.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range arts {
		got = append(got, a.ExternalID+" | "+a.Title+" | "+a.IssuerHint)
	}
	want := []string{
		"1475900 | Karl Thedéen proposed as new Chair of the Board of Net Insight as Anna Söderblom declines re-election | Net Insight AB",
		"1475587 | Saab receives order to equip Sweden's new frigates | SAAB AB",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}

// Other markets and announcements without headline, readable time, issuer
// or link are skipped.
func TestParseSkipsIncompleteItems(t *testing.T) {
	arts, err := nasdaqnordic.ParseNews(read(t, "edge.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 1 || arts[0].ExternalID != "9006" || arts[0].Title != "Net Insight: padded headline" {
		t.Errorf("got %+v", arts)
	}
}

func TestParseEmpty(t *testing.T) {
	for _, parse := range []ingest.Parser{nasdaqnordic.ParseAdhoc, nasdaqnordic.ParseFilings, nasdaqnordic.ParseNews} {
		arts, err := parse(read(t, "empty.json"))
		if err != nil || len(arts) != 0 {
			t.Errorf("arts = %v, err = %v", arts, err)
		}
	}
}

func TestParseBrokenIsError(t *testing.T) {
	for _, b := range [][]byte{read(t, "broken.json"), []byte("<html>"), []byte(`{"error":"busy"}`)} {
		if _, err := nasdaqnordic.ParseAdhoc(b); err == nil {
			t.Errorf("%q: want error", b)
		}
	}
}
