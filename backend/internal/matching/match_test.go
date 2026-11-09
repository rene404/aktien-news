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
		{ID: erste, Name: "Erste Group", LegalName: "Erste Group Bank AG", LEI: "PQOH26KWDF7CG10L6792",
			Aliases:  []matching.Alias{{Text: "Erste", Kind: "core_name"}},
			Listings: []matching.Listing{{Ticker: "EBS", MIC: "XWBO", ISIN: "AT0000652011"}}},
		{ID: porr, Name: "PORR", LegalName: "PORR AG", LEI: "529900Q4M4K5V7GDEN18",
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
		{"source is the issuer's own channel, company not in title",
			matching.Input{Title: "Installation of the electric arc furnace completed", IssuerCompanyID: porr},
			[]want{{porr, "issuer", 1.00}}},
		{"own-channel issuer plus another company in the title",
			matching.Input{Title: "Auftrag von Erste Group Bank AG", IssuerCompanyID: porr},
			[]want{{porr, "issuer", 1.00}, {erste, "full_name", 0.90}}},
		{"issuer ISIN from an ad-hoc feed",
			matching.Input{Title: "Irgendein Titel", IssuerISIN: "AT0000652011", IssuerGrade: true},
			[]want{{erste, "isin", 0.98}}},
		{"issuer LEI from an ad-hoc or filing feed",
			matching.Input{Title: "Directors' Dealings", IssuerLEI: "PQOH26KWDF7CG10L6792", IssuerGrade: true},
			[]want{{erste, "lei", 0.98}}},
		{"issuer LEI links where the feed's issuer name is unknown (OeKB, no ISIN)",
			matching.Input{Title: "EQS-DD: EGB AG: Kauf", IssuerHint: "EGB AG", IssuerLEI: "PQOH26KWDF7CG10L6792", IssuerGrade: true},
			[]want{{erste, "lei", 0.98}}},
		{"unknown LEI changes nothing",
			matching.Input{Title: "Erste erhöht Dividende", IssuerLEI: "5299000000000000000X", IssuerGrade: true},
			[]want{{erste, "core_name", 0.55}}},
		{"LEI from a news feed is no issuer signal",
			matching.Input{Title: "Aktien Wien Schluss", IssuerLEI: "PQOH26KWDF7CG10L6792"},
			nil},
		{"issuer name beats LEI",
			matching.Input{Title: "x", IssuerHint: "PORR AG", IssuerLEI: "529900Q4M4K5V7GDEN18", IssuerGrade: true},
			[]want{{porr, "issuer", 1.00}}},
		{"tagged ISIN from a news feed, company not in title: review",
			matching.Input{Title: "Aktien Wien Schluss: ATX legt zu", IssuerISIN: "AT0000652011"},
			[]want{{erste, "isin", 0.60}}},
		{"tagged ISIN corroborated by core name: linked",
			matching.Input{Title: "Erste erhöht Dividende", IssuerISIN: "AT0000652011"},
			[]want{{erste, "isin", 0.90}}},
		{"tagged ISIN corroborated by ticker: linked",
			matching.Input{Title: "Kursziel für SAP gesenkt", IssuerISIN: "DE0007164600"},
			[]want{{sap, "ticker", 0.90}}}, // tie with the corroborated ISIN; still linked
		{"tagged ISIN does not corroborate another company",
			matching.Input{Title: "Erste erhöht Dividende", IssuerISIN: "DE0007164600"},
			[]want{{sap, "isin", 0.60}, {erste, "core_name", 0.55}}},
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

// Companies imported in bulk from the SEC ticker list are not curated: their
// short tickers and bare names are too ambiguous for titles.
func TestMatchUncuratedNeedsStrongSignals(t *testing.T) {
	const (
		agilent = iota + 100
		onsemi
		gartner
		target
		bancorp
	)
	us := func(id int64, name, ticker, mic string) matching.CompanyEntry {
		return matching.CompanyEntry{ID: id, Name: name, LegalName: name, Uncurated: true,
			Listings: []matching.Listing{{Ticker: ticker, MIC: mic}}}
	}
	ix := matching.NewIndex([]matching.CompanyEntry{
		us(agilent, "Agilent Technologies, Inc.", "A", "XNYS"),
		us(onsemi, "ON SEMICONDUCTOR CORP", "ON", "XNAS"),
		us(bancorp, "Bancorp, Inc.", "TBBK", "XNAS"),
		{ID: gartner, Name: "GARTNER INC", LegalName: "GARTNER INC", CIK: "749251", Uncurated: true,
			Listings: []matching.Listing{{Ticker: "IT", MIC: "XNYS"}}},
		{ID: target, Name: "Target", LegalName: "Target Corporation", Uncurated: true,
			Listings: []matching.Listing{{Ticker: "TGT", MIC: "XNYS", ISIN: "US87612E1064"}}},
	})
	cases := []struct {
		name string
		in   matching.Input
		want []want
	}{
		{"single-letter ticker as a word", matching.Input{Title: "Plan A for ON and IT budgets"}, nil},
		{"ticker word", matching.Input{Title: "TGT shares rise"}, nil},
		{"bare name without legal form", matching.Input{Title: "Target cuts outlook"}, nil},
		{"full name with legal form", matching.Input{Title: "Agilent Technologies, Inc. Reports Q3 Results"},
			[]want{{agilent, "full_name", 0.90}}},
		{"full name inside a longer company name",
			matching.Input{Title: "8-K: Monroe Federal Bancorp, Inc. – Item 5.02 Departure of Directors"}, nil},
		{"full name inside a longer name with a suffix",
			matching.Input{Title: "8-K: Orange County Bancorp, Inc. /DE/ – Item 2.02 Results"}, nil},
		{"full name after a lowercase word", matching.Input{Title: "Shares of Bancorp, Inc. rise"},
			[]want{{bancorp, "full_name", 0.90}}},
		{"full name after a distribution prefix", matching.Input{Title: "EQS-News: Agilent Technologies, Inc. raises outlook"},
			[]want{{agilent, "full_name", 0.90}}},
		{"full name at the start", matching.Input{Title: "Bancorp, Inc. Reports Q3 Results"},
			[]want{{bancorp, "full_name", 0.90}}},
		{"title case: function word before the name", matching.Input{Title: "Shares Of Agilent Technologies, Inc. Rise After Earnings"},
			[]want{{agilent, "full_name", 0.90}}},
		{"title case: 'The' before the name", matching.Input{Title: "Why The Bancorp, Inc. Beat Estimates"},
			[]want{{bancorp, "full_name", 0.90}}},
		{"German function word before the name", matching.Input{Title: "Aktie Von Agilent Technologies, Inc. Steigt"},
			[]want{{agilent, "full_name", 0.90}}},
		{"cashtag", matching.Input{Title: "Why $ON is moving"}, []want{{onsemi, "cashtag", 0.95}}},
		{"filer CIK, title without the name", matching.Input{Title: "8-K: GARTNER INC – Item 8.01 Other Events", IssuerCIK: "749251"},
			[]want{{gartner, "issuer", 1.00}}},
		{"unknown CIK", matching.Input{Title: "8-K: SMALLCAP WIDGETS CORP – Item 8.01 Other Events", IssuerCIK: "1999999"}, nil},
		{"issuer hint", matching.Input{Title: "Q3 results", IssuerHint: "Gartner, Inc."}, []want{{gartner, "issuer", 1.00}}},
		{"ISIN in title", matching.Input{Title: "US87612E1064: dividend"}, []want{{target, "isin", 0.98}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ix.Match(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
			for i, w := range tc.want {
				if g := got[i]; g.CompanyID != w.id || g.Method != w.method || g.Confidence != w.conf {
					t.Errorf("match %d = %+v, want %+v", i, g, w)
				}
			}
		})
	}
}

// Curated names keep matching inside longer names; the containment rule for
// other companies' names covers them.
func TestMatchCuratedFullNameAfterCapitalizedWord(t *testing.T) {
	got := testIndex().Match(matching.Input{Title: "Kurs Erste Group Bank AG steigt"})
	if len(got) != 1 || got[0].CompanyID != erste || got[0].Method != "full_name" {
		t.Errorf("got %+v, want Erste full_name", got)
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
		"A.P. Møller - Mærsk A/S": "a p moller maersk a s",
		"AT&S":                    "at&s",
		"AT & S Austria":          "at s austria",
	}
	for in, want := range cases {
		if got := matching.Fold(in); got != want {
			t.Errorf("Fold(%q) = %q, want %q", in, got, want)
		}
	}
}

// docs/plan-eu-sources.md, task 3: Euronext and Madrid tickers are mnemonics
// ("EN", "RED", "NN") that headlines use as words, not as symbols. They count
// as plain words only where the ticker is also the company's name ("IAG").
// Real titles from the dev database, 2026-11-08.
func TestEUTickersCountOnlyWhereTheyAreTheName(t *testing.T) {
	const (
		bouygues = iota + 100
		redeia
		nn
		iag
	)
	ix := matching.NewIndex([]matching.CompanyEntry{
		{ID: bouygues, Name: "Bouygues", LegalName: "BOUYGUES", Listings: []matching.Listing{{Ticker: "EN", MIC: "XPAR", ISIN: "FR0000120503"}}},
		{ID: redeia, Name: "Redeia", LegalName: "REDEIA CORPORACION SA", Listings: []matching.Listing{{Ticker: "RED", MIC: "XMAD", ISIN: "ES0173093024"}}},
		{ID: nn, Name: "NN Group", LegalName: "NN Group N.V.", Listings: []matching.Listing{{Ticker: "NN", MIC: "XAMS", ISIN: "NL0010773842"}}},
		{ID: iag, Name: "IAG", LegalName: "INTERNATIONAL CONSOLIDATED AIRLINES GROUP, S.A.", Listings: []matching.Listing{{Ticker: "IAG", MIC: "XMAD", ISIN: "ES0177542018"}}},
	})
	cases := []struct {
		title string
		want  []want
	}{
		{"LOGICALIS DEVIENT LE PREMIER PARTENAIRE MONDIAL DE CISCO EN MATIÈRE DE RÉSEAUX", nil},
		{"RED ROBIN ANNOUNCES PRELIMINARY THIRD QUARTER COMPARABLE RESTAURANT REVENUE", nil},
		{"8-K: NN INC – Item 2.02 Results of Operations and Financial Condition", nil},
		{"RBC stuft IAG auf 'Outperform'", []want{{iag, "ticker", 0.90}}},
		{"Why $EN is up today", []want{{bouygues, "cashtag", 0.95}}},
	}
	for _, tc := range cases {
		got := ix.Match(matching.Input{Title: tc.title})
		if len(got) != len(tc.want) {
			t.Errorf("%q: got %+v, want %+v", tc.title, got, tc.want)
			continue
		}
		for i, w := range tc.want {
			if got[i].CompanyID != w.id || got[i].Method != w.method || got[i].Confidence != w.conf {
				t.Errorf("%q: got %+v, want %+v", tc.title, got[i], w)
			}
		}
	}
}

// Regression: Airbus trades as AIR on Xetra and Paris. As a plain word it
// linked "AIR FRANCE-KLM", "AIR LIQUIDE" and "DELTA AIR LINES" (dev database,
// 2026-11-08). Headlines name Airbus anyway; "$AIR" still counts.
func TestTickerAIRIsAWord(t *testing.T) {
	const airbus = 200
	ix := matching.NewIndex([]matching.CompanyEntry{
		{ID: airbus, Name: "Airbus", LegalName: "Airbus SE", Listings: []matching.Listing{
			{Ticker: "AIR", MIC: "XETR", ISIN: "NL0000235190"}, {Ticker: "AIR", MIC: "XPAR", ISIN: "NL0000235190"}}},
	})
	for _, title := range []string{
		"JPMORGAN stuft AIR FRANCE-KLM auf 'Overweight'",
		"BERNSTEIN RESEARCH stuft AIR LIQUIDE(L) auf 'Outperform'",
		"8-K: DELTA AIR LINES, INC. – Item 1.01 Entry into a Material Definitive Agreement",
	} {
		if got := ix.Match(matching.Input{Title: title}); len(got) != 0 {
			t.Errorf("%q linked %+v", title, got)
		}
	}
	if got := ix.Match(matching.Input{Title: "Why $AIR is up today"}); len(got) != 1 || got[0].Method != "cashtag" {
		t.Errorf("cashtag: got %+v", got)
	}
}

// Nasdaq Nordic names its issuers "Volvo, AB", "Tieto Oyj", "Essity
// Aktiebolag (publ)"; Nordic tickers ("SAND", "ISS") are words in headlines.
func TestNordicIssuersAndTickers(t *testing.T) {
	const (
		volvo = iota + 300
		tieto
		essity
		ericsson
		iss
		sandvik
		nokia
		eqt
		nextPLC
		babcock
	)
	ix := matching.NewIndex([]matching.CompanyEntry{
		{ID: volvo, Name: "Volvo", LegalName: "Aktiebolaget Volvo", Listings: []matching.Listing{{Ticker: "VOLV-B", MIC: "XSTO"}}},
		{ID: tieto, Name: "Tieto", LegalName: "Tieto Oyj", Listings: []matching.Listing{{Ticker: "TIETO", MIC: "XHEL"}}},
		{ID: essity, Name: "Essity", LegalName: "Essity Aktiebolag (publ)", Listings: []matching.Listing{{Ticker: "ESSITY-B", MIC: "XSTO"}}},
		{ID: ericsson, Name: "Ericsson", LegalName: "Telefonaktiebolaget LM Ericsson",
			Aliases:  []matching.Alias{{Text: "Ericsson, Telefonab. L M", Kind: "issuer_name"}},
			Listings: []matching.Listing{{Ticker: "ERIC-B", MIC: "XSTO"}}},
		{ID: iss, Name: "ISS", LegalName: "ISS A/S", Listings: []matching.Listing{{Ticker: "ISS", MIC: "XCSE"}}},
		{ID: sandvik, Name: "Sandvik", LegalName: "Sandvik Aktiebolag", Listings: []matching.Listing{{Ticker: "SAND", MIC: "XSTO"}}},
		{ID: nokia, Name: "Nokia", LegalName: "Nokia Oyj", Listings: []matching.Listing{{Ticker: "NOKIA", MIC: "XHEL"}}},
		{ID: eqt, Name: "EQT", LegalName: "EQT AB", Listings: []matching.Listing{{Ticker: "EQT", MIC: "XSTO"}}},
		{ID: nextPLC, Name: "Next plc", LegalName: "NEXT PLC", Listings: []matching.Listing{{Ticker: "NXT", MIC: "XLON"}}},
		{ID: babcock, Name: "Babcock International", LegalName: "BABCOCK INTERNATIONAL GROUP PLC", Listings: []matching.Listing{{Ticker: "BAB", MIC: "XLON"}}},
	})
	cases := []struct {
		in   matching.Input
		want []want
	}{
		{matching.Input{Title: "Buyback of shares", IssuerHint: "Volvo, AB"}, []want{{volvo, "issuer", 1.00}}},
		{matching.Input{Title: "Managers' transactions", IssuerHint: "Tieto Oyj"}, []want{{tieto, "issuer", 1.00}}},
		{matching.Input{Title: "Interim report", IssuerHint: "Essity Aktiebolag (publ)"}, []want{{essity, "issuer", 1.00}}},
		{matching.Input{Title: "Share buybacks", IssuerHint: "Ericsson, Telefonab. L M"}, []want{{ericsson, "issuer", 1.00}}},
		{matching.Input{Title: "Tieto Oyj: Changes in the Leadership Team"}, []want{{tieto, "full_name", 0.90}}},
		// Only the core name, pending: not shown.
		{matching.Input{Title: "NASA EXTENDS ISS OPERATIONS"}, []want{{iss, "core_name", 0.55}}},
		{matching.Input{Title: "SAND PRICES SURGE IN TEXAS"}, nil},
		{matching.Input{Title: "NOKIA WINS 5G DEAL"}, []want{{nokia, "ticker", 0.90}}},
		// LSE tickers (TIDMs) are mnemonics too.
		{matching.Input{Title: "NXT GEN BATTERIES RAISE $40M"}, nil},
		{matching.Input{Title: "BAB FUND RAISES STAKE"}, nil},
		// EQT Corp (US gas) and EQT AB (Swedish investor) share the name.
		{matching.Input{Title: "8-K: EQT Exeter Real Estate Income Trust, Inc. – Item 7.01"}, []want{{eqt, "core_name", 0.55}}},
	}
	for _, tc := range cases {
		got := ix.Match(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("%+v: got %+v, want %+v", tc.in, got, tc.want)
			continue
		}
		for i, w := range tc.want {
			if got[i].CompanyID != w.id || got[i].Method != w.method || got[i].Confidence != w.conf {
				t.Errorf("%+v: got %+v, want %+v", tc.in, got[i], w)
			}
		}
	}
}

// Regression (dev database, 2026-11-09): Nasdaq Nordic's "Castellum AB" and
// "RTX" (RTX A/S, Copenhagen) linked the SEC imports "Castellum, Inc." and
// "RTX Corp" by their names without legal form. That shortcut is for sources
// with US issuers; the exact name still counts everywhere ("ARGENX SE").
func TestUncuratedIssuerNeedsFullNameOutsideUS(t *testing.T) {
	const (
		castellum = iota + 400
		rtx
		argenx
	)
	ix := matching.NewIndex([]matching.CompanyEntry{
		{ID: castellum, Name: "Castellum, Inc.", Uncurated: true, Listings: []matching.Listing{{Ticker: "CTM", MIC: "XNYS"}}},
		{ID: rtx, Name: "RTX Corp", Uncurated: true, Listings: []matching.Listing{{Ticker: "RTX", MIC: "XNYS"}}},
		{ID: argenx, Name: "ARGENX SE", Uncurated: true, Listings: []matching.Listing{{Ticker: "ARGX", MIC: "XNAS"}}},
	})
	cases := []struct {
		in   matching.Input
		want []want
	}{
		{matching.Input{Title: "Change to Castellum's Executive Management", IssuerHint: "Castellum AB", IssuerGrade: true}, nil},
		{matching.Input{Title: "Share buy-back programme", IssuerHint: "RTX", IssuerGrade: true}, nil},
		{matching.Input{Title: "argenx SE: Eigengeschäft einer Führungskraft", IssuerHint: "argenx SE", IssuerGrade: true},
			[]want{{argenx, "issuer", 1.00}}},
		{matching.Input{Title: "RTX wins Army contract", IssuerHint: "RTX", USIssuers: true}, []want{{rtx, "issuer", 1.00}}},
	}
	for _, tc := range cases {
		got := ix.Match(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("%+v: got %+v, want %+v", tc.in, got, tc.want)
			continue
		}
		for i, w := range tc.want {
			if got[i].CompanyID != w.id || got[i].Method != w.method || got[i].Confidence != w.conf {
				t.Errorf("%+v: got %+v, want %+v", tc.in, got[i], w)
			}
		}
	}
}

// Regression (FCA NSM, 2026-11-09): the issuer named by the feed is put in
// front of the title; that name must not count again as a mention of
// another company. "ICG ENTERPRISE TRUST PLC" is not ICG, "BP CAPITAL
// MARKETS" not BP. Other companies in the rest of the title still count.
func TestIssuerNameInTitleIsNoMention(t *testing.T) {
	const (
		icg = iota + 500
		bp
		iberdrola
	)
	ix := matching.NewIndex([]matching.CompanyEntry{
		{ID: icg, Name: "ICG", LegalName: "ICG PLC", Listings: []matching.Listing{{Ticker: "ICG", MIC: "XLON"}}},
		{ID: bp, Name: "BP", LegalName: "BP P.L.C.", Listings: []matching.Listing{{Ticker: "BP", MIC: "XLON"}}},
		{ID: iberdrola, Name: "Iberdrola", LegalName: "IBERDROLA, S.A.", Listings: []matching.Listing{{Ticker: "IBE", MIC: "XMAD"}}},
	})
	cases := []struct {
		in   matching.Input
		want []want
	}{
		{matching.Input{Title: "ICG ENTERPRISE TRUST PLC: Transaction in Own Shares", IssuerHint: "ICG ENTERPRISE TRUST PLC", IssuerGrade: true}, nil},
		{matching.Input{Title: "BP CAPITAL MARKETS P.L.C.: Publication of a Supplementary Prospectus", IssuerHint: "BP CAPITAL MARKETS P.L.C.", IssuerGrade: true}, nil},
		{matching.Input{Title: "ACME SA: Agreement with Iberdrola, S.A.", IssuerHint: "ACME SA", IssuerGrade: true}, []want{{iberdrola, "full_name", 0.90}}},
		{matching.Input{Title: "BP raises dividend"}, []want{{bp, "ticker", 0.90}}},
	}
	for _, tc := range cases {
		got := ix.Match(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("%+v: got %+v, want %+v", tc.in, got, tc.want)
			continue
		}
		for i, w := range tc.want {
			if got[i].CompanyID != w.id || got[i].Method != w.method || got[i].Confidence != w.conf {
				t.Errorf("%+v: got %+v, want %+v", tc.in, got[i], w)
			}
		}
	}
}

// "AT&S" folded to "at s", and the "s" counted as a legal form (S.A.): the
// brand was a full name and matched "at S-Thetics". A glued "&" now stays in
// the folded name. The uppercase ticker of the feed's own issuer is no mention
// of another company either: "6-K: ATS Corp /ATS" is not about AT&S.
func TestAmpersandNamesAndIssuerTickers(t *testing.T) {
	const (
		ats = iota + 600
		atsCorp
		maplebear
	)
	ix := matching.NewIndex([]matching.CompanyEntry{
		{ID: ats, Name: "AT&S", LegalName: "AT & S Austria Technologie & Systemtechnik Aktiengesellschaft",
			Aliases: []matching.Alias{{Text: "AT&S", Kind: "brand"},
				{Text: "AT&S Austria Technologie & Systemtechnik AG", Kind: "issuer_name"}},
			Listings: []matching.Listing{{Ticker: "ATS", MIC: "XWBO", ISIN: "AT0000969985"}}},
		{ID: atsCorp, Name: "ATS Corp", Uncurated: true, CIK: "1898109",
			Listings: []matching.Listing{{Ticker: "ATS", MIC: "XNYS"}}},
		{ID: maplebear, Name: "Maplebear Inc.", Uncurated: true, Listings: []matching.Listing{{Ticker: "CART", MIC: "XNAS"}}},
	})
	cases := []struct {
		in   matching.Input
		want []want
	}{
		{matching.Input{Title: "Meet us at S-Thetics in Vienna"}, nil},
		{matching.Input{Title: "Carrier to Present at Baird's 2026 Global Industrial Conference"}, nil},
		// The brand alone is a core name; FN's ISIN tag confirms it.
		{matching.Input{Title: "Uniper putzt sich raus, Daimler Truck liefert nicht genug und AT&S wird trotz Rekordzahlen abgestraft"},
			[]want{{ats, "core_name", 0.55}}},
		{matching.Input{Title: "Uniper putzt sich raus, Daimler Truck liefert nicht genug und AT&S wird trotz Rekordzahlen abgestraft", IssuerISIN: "AT0000969985"},
			[]want{{ats, "isin", 0.90}}},
		// "M&A" is one word: "Maplebear Inc." follows "of", not "Investigation".
		{matching.Input{Title: "SHAREHOLDER ALERT: The M&A Class Action Firm Announces An Investigation of Maplebear Inc. (NASDAQ: CART)"},
			[]want{{maplebear, "full_name", 0.90}}},
		{matching.Input{Title: "x", IssuerHint: "AT&S Austria Technologie & Systemtechnik AG"}, []want{{ats, "issuer", 1.00}}},
		{matching.Input{Title: "x", IssuerHint: "AT & S Austria Technologie & Systemtechnik Aktiengesellschaft"}, []want{{ats, "issuer", 1.00}}},
		{matching.Input{Title: "ATS hebt Prognose an"}, []want{{ats, "ticker", 0.90}}},
		{matching.Input{Title: "6-K: ATS Corp /ATS", IssuerCIK: "1898109", IssuerGrade: true}, []want{{atsCorp, "issuer", 1.00}}},
	}
	for _, tc := range cases {
		got := ix.Match(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("%+v: got %+v, want %+v", tc.in, got, tc.want)
			continue
		}
		for i, w := range tc.want {
			if got[i].CompanyID != w.id || got[i].Method != w.method || got[i].Confidence != w.conf {
				t.Errorf("%+v: got %+v, want %+v", tc.in, got[i], w)
			}
		}
	}
}
