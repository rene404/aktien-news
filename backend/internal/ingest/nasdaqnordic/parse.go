// Package nasdaqnordic parses the company announcements of Nasdaq Nordic
// (api.news.eu.nasdaq.com, JSON), queried for the main markets and with
// timeZone=UTC. One feed, three sources split by category: inside
// information (ad hoc), the other regulatory announcements (filings) and
// investor news (the issuers' own press releases).
//
// The terms of use of the API are not published; a written request to Nasdaq
// is due before the site goes public (docs/plan-eu-sources.md, task 8).
package nasdaqnordic

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

// markets: the exchanges of the OMXS30, OMXH25 and OMXC25. Iceland is in the
// main-market feed too.
var markets = map[string]bool{
	"Main Market, Stockholm":  true,
	"Main Market, Helsinki":   true,
	"Main Market, Copenhagen": true,
}

const (
	insideInformation = "inside information"
	investorNews      = "investor news"
	managers          = "managers' transactions"
)

// titleDD replaces the headline of managers' transactions, which names the
// person (docs/plan-eu-sources.md, decision 2).
const titleDD = "Eigengeschäft einer Führungskraft"

// twinWindow: issuers publish the English and the local version as two
// announcements, seconds to minutes apart.
const twinWindow = 10 * time.Minute

type feed struct {
	Results *struct {
		Items []item `json:"item"`
	} `json:"results"`
}

type item struct {
	ID       json.Number `json:"disclosureId"`
	Headline string      `json:"headline"`
	Company  string      `json:"company"`
	Market   string      `json:"market"`
	Category string      `json:"cnsCategory"`
	Language string      `json:"language"`
	Released string      `json:"releaseTime"` // "2026-11-09 09:36:36", UTC
	URL      string      `json:"messageUrl"`
}

// ParseAdhoc keeps the inside information (Art. 17 MAR).
func ParseAdhoc(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(c string) bool { return c == insideInformation })
}

// ParseFilings keeps the other regulatory announcements.
func ParseFilings(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(c string) bool { return c != insideInformation && c != investorNews })
}

// ParseNews keeps the investor news.
func ParseNews(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(c string) bool { return c == investorNews })
}

func category(it item) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(it.Category), "’", "'"))
}

func parse(b []byte, keep func(string) bool) ([]ingest.RawArticle, error) {
	var f feed
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("nasdaqnordic: decode: %w", err)
	}
	if f.Results == nil {
		return nil, errors.New("nasdaqnordic: no results")
	}

	type dated struct {
		item
		at time.Time
	}
	var all []dated
	for _, it := range f.Results.Items {
		at, err := time.Parse(time.DateTime, strings.TrimSpace(it.Released))
		if err != nil || !markets[it.Market] || it.ID.String() == "" || strings.TrimSpace(it.Headline) == "" ||
			strings.TrimSpace(it.Company) == "" || strings.TrimSpace(it.URL) == "" {
			continue
		}
		all = append(all, dated{it, at.UTC()})
	}

	arts := make([]ingest.RawArticle, 0, len(all))
	for _, d := range all {
		cat := category(d.item)
		if !keep(cat) {
			continue
		}
		twin := false
		if !strings.EqualFold(d.Language, "en") {
			for _, o := range all {
				if strings.EqualFold(o.Language, "en") && o.Company == d.Company && category(o.item) == cat &&
					o.at.Sub(d.at).Abs() <= twinWindow {
					twin = true
					break
				}
			}
		}
		if twin {
			continue
		}
		issuer := strings.TrimSpace(d.Company)
		title := withIssuer(strings.TrimSpace(d.Headline), issuer)
		if cat == managers {
			title = issuer + ": " + titleDD
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  d.ID.String(),
			URL:         strings.TrimSpace(d.URL),
			Title:       title,
			PublishedAt: d.at,
			IssuerHint:  issuer,
		})
	}
	return arts, nil
}

// withIssuer puts the issuer before a title that does not name it, judged by
// the first word of its name: "Ericsson, Telefonab. L M" → "ericsson".
func withIssuer(title, issuer string) string {
	words := strings.FieldsFunc(strings.ToLower(issuer), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) > 0 && strings.Contains(strings.ToLower(title), words[0]) {
		return title
	}
	return issuer + ": " + title
}
