// Package pressrss parses press-release RSS feeds whose items name the
// sender: APA-OTS in <author>, PR Newswire in <dc:contributor>. The sender
// becomes the issuer hint. The <description> is release text and is ignored.
package pressrss

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

type rss struct {
	Items []item `xml:"channel>item"`
}

type item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
	Contributor string `xml:"http://purl.org/dc/elements/1.1/ contributor"`
}

// Parse extracts articles from a feed. Items without title, link or a
// parseable date are skipped; a malformed document is an error.
func Parse(b []byte) ([]ingest.RawArticle, error) {
	var doc rss
	if err := xml.NewDecoder(bytes.NewReader(b)).Decode(&doc); err != nil {
		return nil, fmt.Errorf("pressrss: decode feed: %w", err)
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
		issuer := strings.TrimSpace(it.Author)
		if issuer == "" {
			issuer = strings.TrimSpace(it.Contributor)
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  id,
			URL:         link,
			Title:       title,
			PublishedAt: published,
			IssuerHint:  issuer,
		})
	}
	return arts, nil
}
