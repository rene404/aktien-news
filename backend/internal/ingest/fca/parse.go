// Package fca reads the FCA National Storage Mechanism (NSM), the UK's
// officially appointed mechanism for regulated information. The search API
// (api.data.fca.org.uk/search?index=nsm-search) takes a JSON POST and answers
// with Elasticsearch hits; Fetcher does the POST, Parse stays pure.
//
// data.fca.org.uk declares "Content-Signal: search=yes, ai-train=no,
// use=reference" in its robots.txt: we only show title and link, and the
// source has text_policy no_llm (docs/plan-eu-sources.md, decision 3).
package fca

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

// ArtefactBase + download_link is the announcement as filed. We only link.
const ArtefactBase = "https://data.fca.org.uk/artefacts/"

// noise are NSM type codes that are not about the issuer's own business:
// fund prices, takeover dealing disclosures by third parties, debt final
// terms and rate fixes, exchange and panel notices.
var noise = map[string]bool{
	"NAV": true, // Net Asset Value(s)
	"RET": true, // Form 8.3
	"FER": true, // Form 8.5 (EPT/RI)
	"FEO": true, // Form 8.5 (EPT/NON-RI)
	"DCC": true, // Form 8 (DD)
	"ITP": true, // Irish Takeover Panel
	"TAB": true, // Disclosure Table (Takeover Panel)
	"PFT": true, // Final Terms
	"RC":  true, // FRN Variable Rate Fix
	"STA": true, // Stabilisation Notice
	"CBO": true, // Cboe Markets
	"NEX": true, // Aquis Exchange
	"ISE": true, // Euronext Dublin
	"SEN": true, // London Stock Exchange Notice
	"AMO": true, // AIM Notice
	"NOT": true, // Official List Notice (FCA)
	"EXC": true, // Exchange Suspension/Restoration
}

// managers is the type code of directors'/PDMR dealings; their title names
// the issuer only (plan decision 2).
const managers = "DSH"

const titleDD = "Eigengeschäft einer Führungskraft"

type response struct {
	Hits *struct {
		Hits []struct {
			Source entry `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type entry struct {
	ID        string `json:"disclosure_id"`
	Headline  string `json:"headline"`
	Company   string `json:"company"`
	LEI       string `json:"lei"`
	TypeCode  string `json:"type_code"`
	Published string `json:"publication_date"`
	Link      string `json:"download_link"`
	Latest    string `json:"latest_flag"`
}

// Parse extracts the announcements. Entries without id, headline, company,
// link or a readable time are skipped; an answer without hits is an error.
func Parse(b []byte) ([]ingest.RawArticle, error) {
	var r response
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("fca: decode: %w", err)
	}
	if r.Hits == nil {
		return nil, errors.New("fca: no hits")
	}
	arts := make([]ingest.RawArticle, 0, len(r.Hits.Hits))
	for _, h := range r.Hits.Hits {
		e := h.Source
		id := strings.TrimSpace(e.ID)
		headline := strings.TrimSpace(e.Headline)
		company := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(e.Company), ";"))
		link := strings.TrimSpace(e.Link)
		at, err := time.Parse(time.RFC3339, strings.TrimSpace(e.Published))
		code := strings.TrimSpace(e.TypeCode)
		if err != nil || noise[code] || e.Latest != "Y" || id == "" || headline == "" || company == "" || link == "" {
			continue
		}
		title := withIssuer(headline, company)
		if code == managers {
			title = company + ": " + titleDD
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  id,
			URL:         ArtefactBase + strings.TrimPrefix(link, "/"),
			Title:       title,
			PublishedAt: at.UTC(),
			IssuerHint:  company,
			IssuerLEI:   strings.ToUpper(strings.TrimSpace(e.LEI)),
		})
	}
	return arts, nil
}

// withIssuer puts the issuer before a headline that does not name it, judged
// by the first word of its name.
func withIssuer(headline, issuer string) string {
	words := strings.FieldsFunc(strings.ToLower(issuer), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) > 0 && strings.Contains(strings.ToLower(headline), words[0]) {
		return headline
	}
	return issuer + ": " + headline
}
