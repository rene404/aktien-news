// Package matching links articles to companies with a confidence per company.
//
// Signals (highest per company wins):
//
//	issuer     1.00  issuer named by the feed/title equals a known name
//	isin       0.98  ISIN supplied by the feed or found in the title
//	cashtag    0.95  $TICKER
//	ticker     0.90  known ticker as an uppercase whole word, not stoplisted
//	full_name  0.90  name incl. legal form (legal name, issuer/former names)
//	core_name  0.55  name without legal form, >= 3 chars → review queue
//
// >= HighThreshold → linked, >= MinThreshold → pending, else dropped.
package matching

import (
	"regexp"
	"sort"
	"strings"
)

const (
	HighThreshold = 0.85
	MinThreshold  = 0.40
)

type Alias struct {
	Text string
	Kind string // core_name | former_name | brand | issuer_name
}

type Listing struct {
	Ticker string
	MIC    string
	ISIN   string
}

type CompanyEntry struct {
	ID        int64
	Name      string
	LegalName string
	Aliases   []Alias
	Listings  []Listing
}

// Input is what the matcher may look at: titles and feed metadata only.
type Input struct {
	Title      string
	IssuerHint string
	IssuerISIN string
}

type Match struct {
	CompanyID  int64
	Confidence float64
	Method     string // matches the match_method enum
}

// Uppercase words that look like tickers but are not meant as one.
var tickerStoplist = map[string]bool{
	"AG": true, "SE": true, "SA": true, "NV": true, "PLC": true, "INC": true, "KGAA": true, "GMBH": true,
	"CEO": true, "CFO": true, "COO": true, "CTO": true, "IR": true, "PR": true, "IT": true, "TV": true,
	"EU": true, "US": true, "USA": true, "UK": true, "DE": true, "AT": true, "CH": true,
	"DAX": true, "MDAX": true, "SDAX": true, "TECDAX": true, "ATX": true, "ETF": true, "IPO": true,
	"ESG": true, "KI": true, "AI": true, "EUR": true, "USD": true, "CHF": true, "EPS": true,
	"EBIT": true, "EBITDA": true, "FY": true, "H1": true, "H2": true, "Q1": true, "Q2": true,
	"Q3": true, "Q4": true, "POS": true, "EQS": true, "DD": true, "OTS": true, "APA": true,
}

var (
	cashtagRe = regexp.MustCompile(`\$([A-Z][A-Z0-9]{0,5}(?:\.[A-Z])?)\b`)
	tickerRe  = regexp.MustCompile(`\b[A-Z][A-Z0-9]{1,5}(?:\.[A-Z])?\b`)
	isinRe    = regexp.MustCompile(`\b[A-Z]{2}[A-Z0-9]{9}[0-9]\b`)
)

type nameEntry struct {
	folded    string
	companyID int64
	conf      float64
	method    string
}

// Index is an immutable lookup structure built from the catalog.
type Index struct {
	issuer  map[string]int64 // folded name, also without legal form → company (-1 = ambiguous)
	isin    map[string]int64
	tickers map[string]int64 // -1 = ambiguous
	names   []nameEntry
}

func NewIndex(companies []CompanyEntry) *Index {
	ix := &Index{issuer: map[string]int64{}, isin: map[string]int64{}, tickers: map[string]int64{}}
	addUnique := func(m map[string]int64, key string, id int64) {
		if key == "" {
			return
		}
		if prev, ok := m[key]; ok && prev != id {
			m[key] = -1
			return
		}
		m[key] = id
	}
	addName := func(text string, id int64, fullOnly bool) {
		f := Fold(text)
		if f == "" {
			return
		}
		addUnique(ix.issuer, f, id)
		addUnique(ix.issuer, stripLegalForm(f), id)
		switch {
		case hasLegalForm(f) || fullOnly:
			ix.names = append(ix.names, nameEntry{f, id, 0.90, "full_name"})
		case len(f) >= 3:
			ix.names = append(ix.names, nameEntry{f, id, 0.55, "core_name"})
		}
	}

	for _, c := range companies {
		addName(c.Name, c.ID, false)
		addName(c.LegalName, c.ID, false)
		for _, a := range c.Aliases {
			addName(a.Text, c.ID, a.Kind == "issuer_name" || a.Kind == "former_name")
		}
		for _, l := range c.Listings {
			addUnique(ix.tickers, l.Ticker, c.ID)
			addUnique(ix.isin, l.ISIN, c.ID)
		}
	}
	return ix
}

// Match returns at most one match per company (the strongest), ordered by
// confidence desc then company id, dropping anything below MinThreshold.
func (ix *Index) Match(in Input) []Match {
	best := map[int64]Match{}
	consider := func(id int64, conf float64, method string) {
		if id <= 0 {
			return
		}
		if cur, ok := best[id]; !ok || conf > cur.Confidence {
			best[id] = Match{CompanyID: id, Confidence: conf, Method: method}
		}
	}

	if h := Fold(in.IssuerHint); h != "" {
		if id, ok := ix.issuer[h]; ok {
			consider(id, 1.00, "issuer")
		} else if id, ok := ix.issuer[stripLegalForm(h)]; ok {
			consider(id, 1.00, "issuer")
		}
	}

	if in.IssuerISIN != "" {
		consider(ix.isin[in.IssuerISIN], 0.98, "isin")
	}
	for _, isin := range isinRe.FindAllString(in.Title, -1) {
		consider(ix.isin[isin], 0.98, "isin")
	}

	for _, m := range cashtagRe.FindAllStringSubmatch(in.Title, -1) {
		consider(ix.tickers[m[1]], 0.95, "cashtag")
	}
	for _, tok := range tickerRe.FindAllString(in.Title, -1) {
		if !tickerStoplist[tok] {
			consider(ix.tickers[tok], 0.90, "ticker")
		}
	}

	for _, m := range ix.nameMatches(Fold(in.Title)) {
		consider(m.companyID, m.conf, m.method)
	}

	out := make([]Match, 0, len(best))
	for _, m := range best {
		if m.Confidence >= MinThreshold {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].CompanyID < out[j].CompanyID
	})
	return out
}

type span struct {
	entry      nameEntry
	start, end int
}

// nameMatches finds every whole-word occurrence of every name and drops
// occurrences that sit inside a longer occurrence of another company's name
// ("Siemens" inside "Siemens Energy").
func (ix *Index) nameMatches(title string) []nameEntry {
	padded := " " + title + " "
	var spans []span
	for _, e := range ix.names {
		needle := " " + e.folded + " "
		for from := 0; ; {
			i := strings.Index(padded[from:], needle)
			if i < 0 {
				break
			}
			start := from + i
			spans = append(spans, span{e, start, start + len(needle)})
			from = start + 1
		}
	}

	var out []nameEntry
	for _, s := range spans {
		contained := false
		for _, o := range spans {
			if o.entry.companyID != s.entry.companyID && o.start <= s.start && s.end <= o.end && o.end-o.start > s.end-s.start {
				contained = true
				break
			}
		}
		if !contained {
			out = append(out, s.entry)
		}
	}
	return out
}

// StatusFor maps a confidence to a link status, or "" if it is dropped.
func StatusFor(conf float64) string {
	switch {
	case conf >= HighThreshold:
		return "linked"
	case conf >= MinThreshold:
		return "pending"
	default:
		return ""
	}
}
