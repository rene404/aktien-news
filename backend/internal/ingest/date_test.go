package ingest_test

import (
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

func TestParseFeedDate(t *testing.T) {
	cases := map[string]time.Time{
		"2026-10-29T11:18:00Z":            time.Date(2026, 10, 29, 11, 18, 0, 0, time.UTC),
		"2026-10-29T07:24:20-04:00":       time.Date(2026, 10, 29, 11, 24, 20, 0, time.UTC),
		"Thu, 29 Oct 2026 12:13:03 +0100": time.Date(2026, 10, 29, 11, 13, 3, 0, time.UTC),
		"Thu, 29 Oct 2026 11:00:02 GMT":   time.Date(2026, 10, 29, 11, 0, 2, 0, time.UTC),
		// RFC 822 allows a one-digit day (Deutsche Telekom's feeds).
		"Mon, 5 Oct 2026 00:00:00 GMT":   time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		"Mon, 5 Oct 2026 09:30:00 +0200": time.Date(2026, 10, 5, 7, 30, 0, 0, time.UTC),
		"  Mon, 5 Oct 2026 00:00:00 GMT ": time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
	}
	for in, want := range cases {
		got, ok := ingest.ParseFeedDate(in)
		if !ok || !got.Equal(want) || got.Location() != time.UTC {
			t.Errorf("ParseFeedDate(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "yesterday", "29.10.2026"} {
		if _, ok := ingest.ParseFeedDate(in); ok {
			t.Errorf("ParseFeedDate(%q) succeeded, want failure", in)
		}
	}
}
