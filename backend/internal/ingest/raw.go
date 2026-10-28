// Package ingest turns source feeds into articles. Parsers in sub-packages are
// pure functions over bytes; networking lives in ingest/fetch.
package ingest

import "time"

// RawArticle is what a parser extracts from one feed item.
//
// Legal invariant: it deliberately has no field for descriptions, snippets or
// body text. Feed descriptions are often third-party agency text (e.g.
// dpa-AFX) that we may neither display nor send to an LLM.
type RawArticle struct {
	ExternalID  string // stable per-source id (guid, or the link if none)
	URL         string
	Title       string
	PublishedAt time.Time
	IssuerHint  string // issuer name stated by the feed or title, if any
	IssuerISIN  string // ISIN supplied by the feed, if any
}
