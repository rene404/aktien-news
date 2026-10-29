package ingest

import (
	"strings"
	"time"
)

var feedDateLayouts = []string{time.RFC3339, time.RFC1123Z, time.RFC1123}

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
