package fca_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/fca"
)

const (
	artefacts = "https://data.fca.org.uk/artefacts/"
	home      = "https://data.fca.org.uk/#/nsm/nationalstoragemechanism"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func utc(s string) time.Time {
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return ts.UTC()
}

// Real NSM entries of 2026-11-09. The NSM may be linked only at its home
// page (terms of use); the announcement's own address is the dedupe key. Fund prices (NAV), takeover dealing
// disclosures (Form 8.3/8.5), final terms and exchange notices are noise;
// superseded versions (latest_flag N) are skipped.
func TestParse(t *testing.T) {
	arts, err := fca.Parse(read(t, "nsm.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{
			ExternalID:  "ef8ce658-4f71-4504-8d11-005af10c6968",
			URL:         home,
			DedupeKey:   artefacts + "NSM/RNS/ef8ce658-4f71-4504-8d11-005af10c6968.html",
			Title:       "ITV PLC: Transaction in Own Shares",
			PublishedAt: utc("2026-11-09T09:27:42Z"),
			IssuerHint:  "ITV PLC",
			IssuerLEI:   "ZLECI7ED2QMWFGYCXZ59",
		},
		{
			// Director/PDMR shareholding: the issuer only (plan decision 2).
			ExternalID:  "5cfa4530-42a3-422b-a0b5-4daf65f5f891",
			URL:         home,
			DedupeKey:   artefacts + "NSM/RNS/5cfa4530-42a3-422b-a0b5-4daf65f5f891.html",
			Title:       "CMC MARKETS PLC: Eigengeschäft einer Führungskraft",
			PublishedAt: utc("2026-11-09T09:26:30Z"),
			IssuerHint:  "CMC MARKETS PLC",
			IssuerLEI:   "213800VB75KAZBFH5U07",
		},
		{
			ExternalID:  "0fb9b3db-5736-4d88-9138-2adbadfae031",
			URL:         home,
			DedupeKey:   artefacts + "NSM/RNS/0fb9b3db-5736-4d88-9138-2adbadfae031.html",
			Title:       "Verici Dx PLC: Holding(s) in Company",
			PublishedAt: utc("2026-11-09T09:36:15Z"),
			IssuerHint:  "Verici Dx PLC",
		},
		{
			ExternalID:  "95b8b96b-ddb1-497a-bb4d-e74eb6b55367",
			URL:         home,
			DedupeKey:   artefacts + "NSM/RNS/95b8b96b-ddb1-497a-bb4d-e74eb6b55367.html",
			Title:       "AFC ENERGY PLC: Trading Update for the year ended 31 October 2026",
			PublishedAt: utc("2026-11-09T07:00:07Z"),
			IssuerHint:  "AFC ENERGY PLC",
			IssuerLEI:   "213800SRJST19GNS9U35",
		},
		{
			// Direct upload: the company ends in ";", the time has milliseconds.
			ExternalID:  "NI-000154371",
			URL:         home,
			DedupeKey:   artefacts + "NSM/DirectUpload/NI-000154371/NI-000154371.pdf",
			Title:       "AUSTRALIA AND NEW ZEALAND BANKING GROUP LIMITED: ANZ 2026 FY Results Presentation & Investor Discussion Pack",
			PublishedAt: utc("2026-11-09T00:15:00Z"),
			IssuerHint:  "AUSTRALIA AND NEW ZEALAND BANKING GROUP LIMITED",
			IssuerLEI:   "JHE42UYNWWTJB8YTTU19",
		},
		{
			ExternalID:  "c76861ec-e805-461e-82a7-022e8db63643",
			URL:         home,
			DedupeKey:   artefacts + "NSM/RNS/c76861ec-e805-461e-82a7-022e8db63643.html",
			Title:       "THAMES WATER UTILITIES LIMITED: Recapitalisation Update",
			PublishedAt: utc("2026-11-09T07:00:08Z"),
			IssuerHint:  "THAMES WATER UTILITIES LIMITED",
			IssuerLEI:   "213800JKM5UQHFJOTZ25",
		},
		{
			ExternalID:  "3bb82d5e-4d09-4f5f-98a9-9e134cf454e4",
			URL:         home,
			DedupeKey:   artefacts + "NSM/PRN/3bb82d5e-4d09-4f5f-98a9-9e134cf454e4.html",
			Title:       "Finsbury Growth & Income Trust PLC - Monthly Fact Sheet as at 31 October 2026",
			PublishedAt: utc("2026-11-09T09:23:00Z"),
			IssuerHint:  "FINSBURY GROWTH & INCOME TRUST PLC",
			IssuerLEI:   "213800NN4ZKX2LG1GQ40",
		},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseEmpty(t *testing.T) {
	arts, err := fca.Parse(read(t, "empty.json"))
	if err != nil || len(arts) != 0 {
		t.Errorf("arts = %v, err = %v", arts, err)
	}
}

func TestParseBrokenIsError(t *testing.T) {
	for _, b := range [][]byte{read(t, "broken.json"), []byte("<html>"), []byte(`{"message":"Missing Authentication Token"}`)} {
		if _, err := fca.Parse(b); err == nil {
			t.Errorf("%q: want error", b)
		}
	}
}
