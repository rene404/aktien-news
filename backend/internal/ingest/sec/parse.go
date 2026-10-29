// Package sec parses the SEC EDGAR "latest filings" Atom feed for 8-K and
// 6-K reports (browse-edgar?action=getcurrent&type=8-K&output=atom). EDGAR data
// is public domain; sec.gov requires a User-Agent with a contact address and
// at most 10 requests per second.
package sec

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

type feed struct {
	Entries []entry `xml:"http://www.w3.org/2005/Atom entry"`
}

type entry struct {
	Title   string `xml:"http://www.w3.org/2005/Atom title"`
	Link    link   `xml:"http://www.w3.org/2005/Atom link"`
	Summary string `xml:"http://www.w3.org/2005/Atom summary"`
	Updated string `xml:"http://www.w3.org/2005/Atom updated"`
	ID      string `xml:"http://www.w3.org/2005/Atom id"`
}

type link struct {
	Href string `xml:"href,attr"`
}

var (
	// "8-K - APPLE INC (0000320193) (Filer)"
	titleRe     = regexp.MustCompile(`^(\S+)\s+-\s+(.+?)\s+\((\d{1,10})\)\s+\(Filer\)\s*$`)
	itemRe      = regexp.MustCompile(`Item (\d+\.\d+):\s*([^<]+)`)
	accessionRe = regexp.MustCompile(`accession-number=([\d-]+)`)
)

// exhibitsItem only lists attachments; it is dropped from titles that name
// other items.
const exhibitsItem = "9.01"

// Parse extracts one article per filing. Only rows for the filer itself
// count; rows without a CIK, link or date are skipped. A malformed document
// is an error.
func Parse(b []byte) ([]ingest.RawArticle, error) {
	var doc feed
	dec := xml.NewDecoder(bytes.NewReader(b))
	dec.CharsetReader = charsetReader
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("sec: decode feed: %w", err)
	}

	arts := make([]ingest.RawArticle, 0, len(doc.Entries))
	for _, e := range doc.Entries {
		m := titleRe.FindStringSubmatch(strings.TrimSpace(e.Title))
		link := strings.TrimSpace(e.Link.Href)
		published, ok := ingest.ParseFeedDate(e.Updated)
		if m == nil || link == "" || !ok {
			continue
		}
		form, name, cik := m[1], m[2], strings.TrimLeft(m[3], "0")
		id := link
		if a := accessionRe.FindStringSubmatch(e.ID); a != nil {
			id = a[1]
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  id,
			URL:         link,
			Title:       title(form, name, e.Summary),
			PublishedAt: published,
			IssuerCIK:   cik,
		})
	}
	return arts, nil
}

// title reads "8-K: APPLE INC – Item 2.02 Results of Operations …" from the
// form, filer and the item list in the summary.
func title(form, name, summary string) string {
	var items []string
	for _, m := range itemRe.FindAllStringSubmatch(summary, -1) {
		items = append(items, "Item "+m[1]+" "+strings.TrimSpace(m[2]))
	}
	if len(items) > 1 {
		kept := items[:0]
		for _, it := range items {
			if !strings.HasPrefix(it, "Item "+exhibitsItem+" ") {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	t := form + ": " + name
	if len(items) > 0 {
		t += " – " + strings.Join(items, " · ")
	}
	return t
}

// charsetReader decodes the ISO-8859-1 EDGAR declares; UTF-8 passes through.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(label) {
	case "iso-8859-1", "latin1", "us-ascii":
		return &latin1Reader{r: bufio.NewReader(input)}, nil
	case "utf-8":
		return input, nil
	}
	return nil, fmt.Errorf("unsupported charset %q", label)
}

type latin1Reader struct {
	r   *bufio.Reader
	buf []byte
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	for len(l.buf) < len(p) {
		b, err := l.r.ReadByte()
		if err != nil {
			if len(l.buf) == 0 {
				return 0, err
			}
			break
		}
		l.buf = utf8.AppendRune(l.buf, rune(b))
	}
	n := copy(p, l.buf)
	l.buf = l.buf[n:]
	return n, nil
}
