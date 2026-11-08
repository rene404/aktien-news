// Package cnmv parses the RSS feeds of the Spanish market regulator CNMV:
// inside information (Información privilegiada) and other relevant
// information (Otra información relevante). Each feed holds the current
// trading day only.
//
// The element names differ from plain RSS (<Channel>, <Title>), and <Title>
// is the issuer. <description> holds time, date, category and the subject of
// the notice, as in
//
//	<p><b>12:37  07/11/2026  </b> Sobre …<BR/><BR/>El Grupo ACS niega …</p>
//
// Only the subject is kept: it is the notice's headline, not body text. The
// CNMV allows reuse "fielmente, sin manipular", so issuer and subject are
// taken over unchanged.
package cnmv

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

type rss struct {
	XMLName xml.Name `xml:"rss"`
	Items   []item   `xml:"Channel>item"`
}

type item struct {
	Title       string `xml:"Title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

var (
	breakRe = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	stampRe = regexp.MustCompile(`^\s*\d{1,2}:\d{2}\s+\d{2}/\d{2}/\d{4}\s*`)
)

// Parse extracts the notices of one feed. Items without issuer, link with
// nreg or a parseable date are skipped; a malformed document is an error.
func Parse(b []byte) ([]ingest.RawArticle, error) {
	var doc rss
	if err := xml.NewDecoder(bytes.NewReader(b)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("cnmv: decode feed: %w", err)
	}

	arts := make([]ingest.RawArticle, 0, len(doc.Items))
	for _, it := range doc.Items {
		issuer := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		nreg := nregOf(link)
		published, ok := ingest.ParseFeedDate(it.PubDate)
		subject := subjectOf(it.Description)
		if issuer == "" || nreg == "" || !ok || subject == "" {
			continue
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  nreg,
			URL:         link,
			Title:       issuer + ": " + subject,
			PublishedAt: published,
			IssuerHint:  issuer,
		})
	}
	return arts, nil
}

func nregOf(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return u.Query().Get("nreg")
}

// subjectOf returns the last part of the description: the subject, or the
// category when a notice has none.
func subjectOf(desc string) string {
	var parts []string
	for _, p := range breakRe.Split(desc, -1) {
		p = strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(p, "")))
		p = strings.TrimSpace(stampRe.ReplaceAllString(p, ""))
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(strings.Fields(parts[len(parts)-1]), " ")
}
