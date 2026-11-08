// Package parsers names every feed parser by its sources.parser key, so the
// worker and the tests share one registry.
package parsers

import (
	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/cnmv"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
	"github.com/rene404/aktien-news/backend/internal/ingest/oekb"
	"github.com/rene404/aktien-news/backend/internal/ingest/pressrss"
	"github.com/rene404/aktien-news/backend/internal/ingest/sec"
)

// All returns a fresh map; callers may change it.
func All() map[string]ingest.Parser {
	return map[string]ingest.Parser{
		"cnmv":              cnmv.Parse,
		"finanznachrichten": finanznachrichten.Parse,
		"oekb_adhoc":        oekb.ParseAdhoc,
		"oekb_filings":      oekb.ParseFilings,
		"pressrss":          pressrss.Parse,
		"sec":               sec.Parse,
	}
}
