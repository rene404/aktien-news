// Package finanznachrichten parses FinanzNachrichten.de RSS feeds.
//
// Terms (finanznachrichten.de/service/rss.htm): headlines with active links may
// be taken over free of charge, fetched at most every 10 minutes, links open
// in a new window, no frames. The <description> is agency text and is ignored.
package finanznachrichten

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

type rss struct {
	Items []item `xml:"channel>item"`
}

// ISIN lives in the fn namespace: xmlns:fn="http://www.finanznachrichten.de/service/rss".
type item struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
	ISIN    string `xml:"http://www.finanznachrichten.de/service/rss isin"`
}

// Parse extracts articles from a feed. Items without title, link or a
// parseable date are skipped; a malformed document is an error.
func Parse(b []byte) ([]ingest.RawArticle, error) {
	var doc rss
	dec := xml.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("finanznachrichten: decode feed: %w", err)
	}

	arts := make([]ingest.RawArticle, 0, len(doc.Items))
	for _, it := range doc.Items {
		title := strings.TrimSpace(it.Title)
		link := strings.TrimSpace(it.Link)
		published, ok := ingest.ParseFeedDate(it.PubDate)
		if title == "" || link == "" || !ok {
			continue
		}
		id := strings.TrimSpace(it.GUID)
		if id == "" {
			id = link
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  id,
			URL:         link,
			Title:       title,
			PublishedAt: published,
			IssuerHint:  IssuerHint(title),
			IssuerISIN:  strings.TrimSpace(it.ISIN),
		})
	}
	return arts, nil
}

// Distribution prefixes of mandatory/corporate announcements, e.g.
// "EQS-Adhoc: Lenzing AG: …", "PTA-DD: BAWAG Group AG: …", "Adhoc: Rieter AG: …".
var issuerTitle = regexp.MustCompile(`^(?:(?:EQS|PTA)-[^:\s]+|Adhoc|euro adhoc[^:]*):\s*([^:]+?):\s`)

// IssuerHint returns the issuer named in an announcement title, or "".
func IssuerHint(title string) string {
	m := issuerTitle.FindStringSubmatch(title)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}
