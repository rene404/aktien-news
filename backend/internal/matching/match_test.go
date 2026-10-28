package matching_test

import (
	"testing"

	"github.com/rene404/aktien-news/backend/internal/matching"
)

const (
	siemens = iota + 1
	siemensEnergy
	post
	erste
	porr
	apple
	sap
	merck
	merckUS
)

func testIndex() *matching.Index {
	return matching.NewIndex([]matching.CompanyEntry{
		{ID: siemens, Name: "Siemens", LegalName: "Siemens Aktiengesellschaft",
			Aliases:  []matching.Alias{{Text: "Siemens", Kind: "core_name"}, {Text: "Siemens AG", Kind: "issuer_name"}},
			Listings: []matching.Listing{{Ticker: "SIE", MIC: "XETR", ISIN: "DE0007236101"}}},
		{ID: siemensEnergy, Name: "Siemens Energy", LegalName: "Siemens Energy AG",
			Aliases:  []matching.Alias{{Text: "Siemens Energy", Kind: "core_name"}},
			Listings: []matching.Listing{{Ticker: "ENR", MIC: "XETR", ISIN: "DE000ENER6Y0"}}},
		{ID: post, Name: "Österreichische Post", LegalName: "Österreichische Post AG",
			Aliases:  []matching.Alias{{Text: "Österreichische Post", Kind: "core_name"}},
			Listings: []matching.Listing{{Ticker: "POST", MIC: "XWBO", ISIN: "AT0000APOST4"}}},
		{ID: erste, Name: "Erste Group", LegalName: "Erste Group Bank AG",
			Aliases:  []matching.Alias{{Text: "Erste", Kind: "core_name"}},
			Listings: []matching.Listing{{Ticker: "EBS", MIC: "XWBO", ISIN: "AT0000652011"}}},
		{ID: porr, Name: "PORR", LegalName: "PORR AG",
			Aliases:  []matching.Alias{{Text: "PORR", Kind: "core_name"}},
			Listings: []matching.Listing{{Ticker: "POS", MIC: "XWBO", ISIN: "AT0000609607"}}},
		{ID: apple, Name: "Apple", LegalName: "Apple Inc.",
			Listings: []matching.Listing{{Ticker: "AAPL", MIC: "XNAS", ISIN: "US0378331005"}}},
		{ID: sap, Name: "SAP", LegalName: "SAP SE",
			Listings: []matching.Listing{{Ticker: "SAP", MIC: "XETR", ISIN: "DE0007164600"}}},
		// Same ticker on two exchanges for two different companies: ambiguous.
		{ID: merck, Name: "Merck", LegalName: "Merck KGaA",
			Aliases:  []matching.Alias{{Text: "Merck", Kind: "core_name"}},
			Listings: []matching.Listing{{Ticker: "MRK", MIC: "XETR", ISIN: "DE0006599905"}}},
		{ID: merckUS, Name: "Merck & Co.", LegalName: "Merck & Co., Inc.",
			Listings: []matching.Listing{{Ticker: "MRK", MIC: "XNYS", ISIN: "US58933Y1055"}, {Ticker: "AG", MIC: "XNYS"}}},
	})
}

type want struct {
	id     int64
	method string
	conf   float64
}

func TestMatch(t *testing.T) {
	ix := testIndex()
	cases := []struct {
		name string
		in   matching.Input
		want []want
	}{
		{"issuer hint from ad-hoc title",
			matching.Input{Title: "EQS-Adhoc: Siemens Energy AG: Prognose angehoben", IssuerHint: "Siemens Energy AG"},
			[]want{{siemensEnergy, "issuer", 1.00}}},
		{"issuer hint with legal form spelled out",
			matching.Input{Title: "x", IssuerHint: "Siemens Aktiengesellschaft"},
			[]want{{siemens, "issuer", 1.00}}},
		{"issuer hint without legal form, umlaut folded",
			matching.Input{Title: "x", IssuerHint: "Oesterreichische Post"},
			[]want{{post, "issuer", 1.00}}},
		{"ISIN from the feed",
			matching.Input{Title: "Irgendein Titel", IssuerISIN: "AT0000652011"},
			[]want{{erste, "isin", 0.98}}},
		{"ISIN in the title",
			matching.Input{Title: "Stimmrechtsmitteilung DE0007164600"},
			[]want{{sap, "isin", 0.98}}},
		{"cashtag",
			matching.Input{Title: "Why $AAPL is up today"},
			[]want{{apple, "cashtag", 0.95}}},
		{"uppercase ticker",
			matching.Input{Title: "SAP verfehlt Erwartungen"},
			[]want{{sap, "ticker", 0.90}}},
		{"full legal name in text",
			matching.Input{Title: "Neuer Vorstand bei der Erste Group Bank AG"},
			[]want{{erste, "full_name", 0.90}}},
		{"core name only is pending-level",
			matching.Input{Title: "Erste erhöht Dividende"},
			[]want{{erste, "core_name", 0.55}}},
		{"longer name wins over contained core name",
			matching.Input{Title: "Siemens Energy schreibt schwarze Zahlen"},
			[]want{{siemensEnergy, "core_name", 0.55}}},
		{"both companies when both are mentioned",
			matching.Input{Title: "Siemens und Siemens Energy im Fokus"},
			[]want{{siemens, "core_name", 0.55}, {siemensEnergy, "core_name", 0.55}}},
		{"highest signal per company wins",
			matching.Input{Title: "EQS-Adhoc: SAP SE: SAP hebt Ausblick", IssuerHint: "SAP SE", IssuerISIN: "DE0007164600"},
			[]want{{sap, "issuer", 1.00}}},
		{"stoplisted words are not tickers",
			matching.Input{Title: "POS-Systeme: CEO der AG tritt zurück"},
			nil},
		{"lowercase ticker-like words are not tickers",
			matching.Input{Title: "Die Post bringt allen was"},
			nil},
		{"ambiguous ticker across companies is ignored",
			matching.Input{Title: "Deal: MRK and AG news"},
			nil},
		{"no match",
			matching.Input{Title: "Ölpreis steigt weiter"},
			nil},
		{"issuer hint for unknown company falls back to text",
			matching.Input{Title: "EQS-Adhoc: Befesa S.A.: Guidance", IssuerHint: "Befesa S.A."},
			nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ix.Match(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			for i, w := range tc.want {
				g := got[i]
				if g.CompanyID != w.id || g.Method != w.method || g.Confidence != w.conf {
					t.Errorf("match %d = %+v, want %+v", i, g, w)
				}
			}
		})
	}
}

func TestStatusFor(t *testing.T) {
	cases := map[float64]string{1.0: "linked", 0.90: "linked", 0.85: "linked", 0.84: "pending", 0.55: "pending", 0.40: "pending", 0.39: ""}
	for conf, want := range cases {
		if got := matching.StatusFor(conf); got != want {
			t.Errorf("StatusFor(%v) = %q, want %q", conf, got, want)
		}
	}
}

func TestFold(t *testing.T) {
	cases := map[string]string{
		"Österreichische Post AG": "oesterreichische post ag",
		"Straße & Co. KGaA":       "strasse co kgaa",
		"  Mercedes-Benz  Group ": "mercedes benz group",
		"Société Générale":        "societe generale",
	}
	for in, want := range cases {
		if got := matching.Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}
