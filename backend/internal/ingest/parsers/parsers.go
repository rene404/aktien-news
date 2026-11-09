// Package parsers names every feed parser by its sources.parser key, so the
// worker and the tests share one registry.
package parsers

import (
	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/afm"
	"github.com/rene404/aktien-news/backend/internal/ingest/cnmv"
	"github.com/rene404/aktien-news/backend/internal/ingest/dila"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
	"github.com/rene404/aktien-news/backend/internal/ingest/nasdaqnordic"
	"github.com/rene404/aktien-news/backend/internal/ingest/oekb"
	"github.com/rene404/aktien-news/backend/internal/ingest/pressrss"
	"github.com/rene404/aktien-news/backend/internal/ingest/sec"
)

// All returns a fresh map; callers may change it.
func All() map[string]ingest.Parser {
	return map[string]ingest.Parser{
		"afm":                   afm.Parse,
		"cnmv":                  cnmv.Parse,
		"dila_adhoc":            dila.ParseAdhoc,
		"dila_filings":          dila.ParseFilings,
		"finanznachrichten":     finanznachrichten.Parse,
		"nasdaq_nordic_adhoc":   nasdaqnordic.ParseAdhoc,
		"nasdaq_nordic_filings": nasdaqnordic.ParseFilings,
		"nasdaq_nordic_news":    nasdaqnordic.ParseNews,
		"oekb_adhoc":            oekb.ParseAdhoc,
		"oekb_filings":          oekb.ParseFilings,
		"pressrss":              pressrss.Parse,
		"sec":                   sec.Parse,
	}
}
