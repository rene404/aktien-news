// Package oekb parses the public list of regulated issuer notices of OeKB
// OAM Issuer Info, Austria's officially appointed mechanism under the
// Transparency Directive (ad-hoc, managers' transactions, voting rights,
// financial reports, AGMs):
//
//	https://my.oekb.at/issuer-info/rest/public/meldedaten/iic?startPosition=0&offset=50
//
// The endpoint backs the OAM web app and is not documented; the OAM terms
// (Nutzungsgrundlagen V5.2) grant end users free query and download access.
// Every notice carries the issuer and the ISINs concerned, so the ISIN is
// issuer-grade.
package oekb

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

// DownloadURL serves a notice's file (PDF). The web app has no page per
// notice, so this is the link.
const DownloadURL = "https://my.oekb.at/issuer-info/rest/public/meldedaten/iic/download?id="

const (
	adhocType = "EP_AD_HOC"
	published = "VEROEFFENTLICHT"
	// twinWindow: the English version of a notice is uploaded within
	// minutes of the German one.
	twinWindow = 10 * time.Minute
)

type response struct {
	Documents *[]document `json:"dokumente"`
}

type document struct {
	ID       int64    `json:"id"`
	Title    string   `json:"titel"`
	Type     string   `json:"meldetypCode"`
	Language string   `json:"sprachcode"`
	Uploaded int64    `json:"uploadzeitpunkt"` // ms since epoch
	Status   string   `json:"status"`
	ISINs    []string `json:"isinBezug"`
	Issuer   struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"emittent"`
	Files []struct {
		ID int64 `json:"id"`
	} `json:"dateien"`
}

// ParseAdhoc keeps the ad-hoc notices (inside information, Art. 17 MAR).
func ParseAdhoc(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(t string) bool { return t == adhocType })
}

// ParseFilings keeps every other notice type.
func ParseFilings(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(t string) bool { return t != adhocType })
}

func parse(b []byte, keep func(string) bool) ([]ingest.RawArticle, error) {
	var resp response
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, fmt.Errorf("oekb: decode: %w", err)
	}
	if resp.Documents == nil {
		return nil, errors.New("oekb: no dokumente in response")
	}
	docs := *resp.Documents

	arts := make([]ingest.RawArticle, 0, len(docs))
	for _, d := range docs {
		title := strings.TrimSpace(d.Title)
		issuer := strings.TrimSpace(d.Issuer.Name)
		if !keep(d.Type) || d.Status != published || len(d.Files) == 0 ||
			title == "" || issuer == "" || d.Uploaded <= 0 || hasGermanTwin(d, docs) {
			continue
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  "oam-" + strconv.FormatInt(d.ID, 10),
			URL:         DownloadURL + strconv.FormatInt(d.Files[0].ID, 10),
			Title:       withIssuer(title, issuer),
			PublishedAt: time.UnixMilli(d.Uploaded).UTC(),
			IssuerHint:  issuer,
			IssuerISIN:  shareISIN(d.ISINs),
		})
	}
	return arts, nil
}

// hasGermanTwin reports whether d is the English version of a German notice
// on the same page. Issuers publish both; the German one is kept, an
// English-only notice stays.
func hasGermanTwin(d document, docs []document) bool {
	if d.Language != "EN" {
		return false
	}
	for _, o := range docs {
		if o.Language == "DE" && o.Issuer.ID == d.Issuer.ID && o.Type == d.Type &&
			absDuration(time.Duration(o.Uploaded-d.Uploaded)*time.Millisecond) <= twinWindow {
			return true
		}
	}
	return false
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// wirePrefix is the distributor's tag, e.g. "PTA-PVR: " or "EQS-DD: ".
var wirePrefix = regexp.MustCompile(`^[A-Z]{2,5}-[A-Za-z]{2,10}: `)

// withIssuer adds the issuer unless the title names it already: many
// notices are titled only by their legal basis ("PTA-PVR: Veröffentlichung
// gemäß § 135 Abs. 2 BörseG"). The first word of the name is the test, so
// "Lenzing AG" counts for "Lenzing Aktiengesellschaft". The issuer goes after
// the distributor's tag, as the wires title it ("PTA-PVR: OMV
// Aktiengesellschaft: …"), so the same notice from a news feed clusters
// with ours.
func withIssuer(title, issuer string) string {
	first, _, _ := strings.Cut(issuer, " ")
	if strings.Contains(strings.ToLower(title), strings.ToLower(first)) {
		return title
	}
	tag := wirePrefix.FindString(title)
	return tag + issuer + ": " + title[len(tag):]
}

var isinRe = regexp.MustCompile(`\b[A-Z]{2}[A-Z0-9]{9}[0-9]\b`)

// shareISIN picks the share from entries like
// "AT0000A21LA8 (Anleihe), AT0000821103 (Aktie),"; without a share, the
// first ISIN.
func shareISIN(entries []string) string {
	all := strings.Join(entries, ",")
	first := ""
	for _, part := range strings.Split(all, ",") {
		isin := isinRe.FindString(part)
		if isin == "" {
			continue
		}
		if strings.Contains(part, "(Aktie)") {
			return isin
		}
		if first == "" {
			first = isin
		}
	}
	return first
}
