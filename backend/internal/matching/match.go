// Package matching links articles to companies with a confidence per company.
//
// Signals (highest per company wins):
//
//	issuer     1.00  issuer named by the feed/title equals a known name, the
//	                 source is the issuer's own channel, or the filer's CIK
//	isin       0.98  issuer ISIN (ad-hoc/filing feed) or ISIN in the title
//	isin       0.90  ISIN tagged by a news feed, company also named in the title
//	isin       0.60  ISIN tagged by a news feed, not corroborated → review
//	cashtag    0.95  $TICKER
//	ticker     0.90  known ticker as an uppercase whole word, not stoplisted
//	full_name  0.90  name incl. legal form (legal name, issuer/former names)
//	core_name  0.55  name without legal form, >= 3 chars → review queue
//
// ticker and core_name do not apply to uncurated (bulk-imported) companies,
// and their full_name does not count right after a capitalized word other
// than a function word ("Of", "The", "Von").
// >= HighThreshold → linked, >= MinThreshold → pending, else dropped.
package matching

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
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
	CIK       string // SEC CIK, no leading zeros
	Aliases   []Alias
	Listings  []Listing
	// Uncurated companies come from a bulk import (SEC ticker list). Only
	// strong signals link them: issuer, ISIN, cashtag, full name with legal
	// form. Their tickers as plain words and bare names are too ambiguous
	// ("A", "ON", "Target").
	Uncurated bool
}

// Input is what the matcher may look at: titles and feed metadata only.
type Input struct {
	Title      string
	IssuerHint string
	IssuerISIN string // ISIN supplied by the feed
	// ISINIsIssuer marks IssuerISIN as the issuer of the item (ad-hoc and
	// filing feeds). In news feeds the ISIN is only a loose tag.
	ISINIsIssuer bool
	// IssuerCompanyID is set when the source is the issuer's own channel
	// (IR feed): every item is about that company.
	IssuerCompanyID int64
	// IssuerCIK is the SEC CIK of the filer (filing feeds).
	IssuerCIK string
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
	uncurated bool
}

// Index is an immutable lookup structure built from the catalog.
type Index struct {
	issuer   map[string]int64 // folded name, also without legal form → company (-1 = ambiguous)
	cik      map[string]int64
	isin     map[string]int64
	cashtags map[string]int64 // every ticker; -1 = ambiguous
	tickers  map[string]int64 // tickers of curated companies, matched as plain words; -1 = ambiguous
	names    []nameEntry
}

func NewIndex(companies []CompanyEntry) *Index {
	ix := &Index{issuer: map[string]int64{}, cik: map[string]int64{}, isin: map[string]int64{}, cashtags: map[string]int64{}, tickers: map[string]int64{}}
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
	addName := func(text string, id int64, fullOnly, uncurated bool) {
		f := Fold(text)
		if f == "" {
			return
		}
		addUnique(ix.issuer, f, id)
		addUnique(ix.issuer, stripLegalForm(f), id)
		switch {
		case hasLegalForm(f) || fullOnly:
			ix.names = append(ix.names, nameEntry{f, id, 0.90, "full_name", uncurated})
		case len(f) >= 3 && !uncurated:
			ix.names = append(ix.names, nameEntry{f, id, 0.55, "core_name", false})
		}
	}

	for _, c := range companies {
		addUnique(ix.cik, c.CIK, c.ID)
		addName(c.Name, c.ID, false, c.Uncurated)
		addName(c.LegalName, c.ID, false, c.Uncurated)
		for _, a := range c.Aliases {
			addName(a.Text, c.ID, a.Kind == "issuer_name" || a.Kind == "former_name", c.Uncurated)
		}
		for _, l := range c.Listings {
			addUnique(ix.cashtags, l.Ticker, c.ID)
			if !c.Uncurated {
				addUnique(ix.tickers, l.Ticker, c.ID)
			}
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

	consider(in.IssuerCompanyID, 1.00, "issuer")
	if in.IssuerCIK != "" {
		consider(ix.cik[in.IssuerCIK], 1.00, "issuer")
	}
	if h := Fold(in.IssuerHint); h != "" {
		if id, ok := ix.issuer[h]; ok {
			consider(id, 1.00, "issuer")
		} else if id, ok := ix.issuer[stripLegalForm(h)]; ok {
			consider(id, 1.00, "issuer")
		}
	}

	for _, isin := range isinRe.FindAllString(in.Title, -1) {
		consider(ix.isin[isin], 0.98, "isin")
	}

	for _, m := range cashtagRe.FindAllStringSubmatch(in.Title, -1) {
		consider(ix.cashtags[m[1]], 0.95, "cashtag")
	}
	for _, tok := range tickerRe.FindAllString(in.Title, -1) {
		if !tickerStoplist[tok] {
			consider(ix.tickers[tok], 0.90, "ticker")
		}
	}

	words := titleWords(in.Title)
	for _, m := range ix.nameMatches(Fold(in.Title)) {
		// An uncurated name glued to a capitalized word is the tail of a
		// longer company name: "Bancorp, Inc." in "Monroe Federal Bancorp, Inc.".
		if m.entry.uncurated && words.continuesName(m.word) {
			continue
		}
		consider(m.entry.companyID, m.entry.conf, m.entry.method)
	}

	// Feed ISIN last: a news-feed tag only counts as strong evidence when the
	// title itself already points at the same company.
	if id := ix.isin[in.IssuerISIN]; in.IssuerISIN != "" && id > 0 {
		_, named := best[id]
		switch {
		case in.ISINIsIssuer:
			consider(id, 0.98, "isin")
		case named:
			consider(id, 0.90, "isin")
		default:
			consider(id, 0.60, "isin")
		}
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
	word       int // index of the first word of the match in the title
}

// nameMatches finds every whole-word occurrence of every name and drops
// occurrences that sit inside a longer occurrence of another company's name
// ("Siemens" inside "Siemens Energy").
func (ix *Index) nameMatches(title string) []span {
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
			spans = append(spans, span{e, start, start + len(needle), strings.Count(padded[:start], " ")})
			from = start + 1
		}
	}

	var out []span
	for _, s := range spans {
		contained := false
		for _, o := range spans {
			if o.entry.companyID != s.entry.companyID && o.start <= s.start && s.end <= o.end && o.end-o.start > s.end-s.start {
				contained = true
				break
			}
		}
		if !contained {
			out = append(out, s)
		}
	}
	return out
}

// word is one alphanumeric run of the original title. Fold keeps these runs
// one-to-one, so word i here is word i of the folded title.
type word struct {
	start, end int // byte offsets in the original title
}

type titleWordList struct {
	title string
	words []word
}

func titleWords(title string) titleWordList {
	var ws []word
	start := -1
	for i, r := range title {
		alnum := unicode.IsLetter(r) || unicode.IsDigit(r)
		switch {
		case alnum && start < 0:
			start = i
		case !alnum && start >= 0:
			ws = append(ws, word{start, i})
			start = -1
		}
	}
	if start >= 0 {
		ws = append(ws, word{start, len(title)})
	}
	return titleWordList{title, ws}
}

// Words that start a capitalized phrase in Title Case headlines ("Shares Of
// …", "Why The …") or German titles, but never continue a company name.
var functionWords = map[string]bool{
	"a": true, "about": true, "after": true, "an": true, "and": true, "as": true, "at": true,
	"before": true, "by": true, "for": true, "from": true, "how": true, "in": true, "into": true,
	"of": true, "on": true, "or": true, "over": true, "the": true, "to": true, "vs": true,
	"what": true, "why": true, "with": true,
	"am": true, "auf": true, "aus": true, "bei": true, "das": true, "dem": true, "den": true,
	"der": true, "des": true, "die": true, "für": true, "gegen": true, "im": true, "mit": true,
	"nach": true, "oder": true, "über": true, "und": true, "von": true, "vor": true, "warum": true,
	"wie": true, "zu": true, "zum": true, "zur": true,
}

// continuesName reports whether word i directly follows a capitalized word
// other than a function word, separated only by whitespace.
func (l titleWordList) continuesName(i int) bool {
	if i <= 0 || i >= len(l.words) {
		return false
	}
	prev, cur := l.words[i-1], l.words[i]
	if strings.TrimSpace(l.title[prev.end:cur.start]) != "" {
		return false
	}
	text := l.title[prev.start:prev.end]
	r, _ := utf8.DecodeRuneInString(text)
	return unicode.IsUpper(r) && !functionWords[strings.ToLower(text)]
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
