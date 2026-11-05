package ingest

import (
	"strings"
	"time"
)

// RFC 1123 layouts with a one- or two-digit day: RFC 822 allows "5 Oct".
var feedDateLayouts = []string{
	time.RFC3339,
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05 MST",
}

// ParseFeedDate parses an RSS pubDate or Atom timestamp into UTC.
func ParseFeedDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range feedDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
