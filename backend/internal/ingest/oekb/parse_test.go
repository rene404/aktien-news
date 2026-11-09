package oekb_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/oekb"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func ms(v int64) time.Time { return time.UnixMilli(v).UTC() }

const pdf = "https://my.oekb.at/issuer-info/rest/public/meldedaten/iic/download?id="

func TestParseFilings(t *testing.T) {
	arts, err := oekb.ParseFilings(read(t, "iic.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{
			// The English twin a minute later is dropped. The notice does not
			// name the issuer, so it goes after the wire tag, as in news feeds.
			ExternalID:  "oam-253438",
			URL:         pdf + "253437",
			Title:       "PTA-PVR: OMV Aktiengesellschaft: Veröffentlichung gemäß § 135 Abs. 2 BörseG",
			PublishedAt: ms(1793730840000),
			IssuerHint:  "OMV Aktiengesellschaft",
			IssuerISIN:  "AT0000743059",
			IssuerLEI:   "549300V62YJ9HWLOJR90",
		},
		{
			// No ISIN: the issuer's name and LEI are the hints.
			ExternalID:  "oam-253390",
			URL:         pdf + "253389",
			Title:       "EQS-DD: Raiffeisenlandesbank Niederösterreich-Wien AG: Dr. Roland Mechtler, Kauf",
			PublishedAt: ms(1793438400000),
			IssuerHint:  "RAIFFEISENLANDESBANK NIEDERÖSTERREICH-WIEN AG",
			IssuerLEI:   "529900GPOO9ISPD1EE83",
		},
		{
			// The share's ISIN wins over the bonds listed before it.
			ExternalID:  "oam-253380",
			URL:         pdf + "253379",
			Title:       "EQS-HV: UNIQA Insurance Group AG: Ergebnisse der Hauptversammlung",
			PublishedAt: ms(1793430000000),
			IssuerHint:  "UNIQA Insurance Group AG",
			IssuerISIN:  "AT0000821103",
		},
		// Without a file there is nothing to link; a withdrawn notice is not news.
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseAdhoc(t *testing.T) {
	arts, err := oekb.ParseAdhoc(read(t, "iic.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{
			// English only: kept. No wire tag: the issuer leads. The trailing
			// dot is not part of the ISIN.
			ExternalID:  "oam-253420",
			URL:         pdf + "253419",
			Title:       "RHI Magnesita N.V.: Further statement regarding possible offer for Vesuvius plc",
			PublishedAt: ms(1793637660000),
			IssuerHint:  "RHI Magnesita N.V.",
			IssuerISIN:  "NL0012650360",
			IssuerLEI:   "724500F4VP0QUKBUT357",
		},
		{
			// The title names the issuer already (case aside).
			ExternalID:  "oam-253401",
			URL:         pdf + "253400",
			Title:       "PTA-Adhoc: POLYTEC HOLDING AG senkt Ergebnisprognose für das Geschäftsjahr 2026",
			PublishedAt: ms(1793447100000),
			IssuerHint:  "POLYTEC Holding AG",
			IssuerISIN:  "AT0000A00XX9",
		},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseLivePage(t *testing.T) {
	b := read(t, "iic_live.json")
	adhoc, err := oekb.ParseAdhoc(b)
	if err != nil {
		t.Fatal(err)
	}
	filings, err := oekb.ParseFilings(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(adhoc) == 0 || len(filings) == 0 {
		t.Fatalf("adhoc = %d, filings = %d", len(adhoc), len(filings))
	}
	// About half the page are English twins of German notices.
	if n := len(adhoc) + len(filings); n >= 40 {
		t.Errorf("%d of 50 kept; English twins not dropped?", n)
	}
	for _, a := range append(adhoc, filings...) {
		if a.Title == "" || a.URL == "" || a.PublishedAt.IsZero() || a.IssuerHint == "" {
			t.Errorf("incomplete: %+v", a)
		}
		// Every issuer on the live page has a LEI.
		if len(a.IssuerLEI) != 20 {
			t.Errorf("LEI %q of %s", a.IssuerLEI, a.ExternalID)
		}
	}
}

func TestParseBrokenIsError(t *testing.T) {
	for _, b := range []string{"<html>", `{"dokumente":`} {
		if _, err := oekb.ParseFilings([]byte(b)); err == nil {
			t.Errorf("%q: want error", b)
		}
	}
	// A JSON object without the list is not the OAM response either.
	if _, err := oekb.ParseAdhoc([]byte(`{"error":"x"}`)); err == nil {
		t.Error("missing dokumente: want error")
	}
}
