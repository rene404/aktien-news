package fca

import (
	"context"
	"fmt"
)

// PageSize: the newest entries per poll. The NSM gets roughly 200 entries an
// hour at the morning peak, most of them fund prices; polled every 15 minutes,
// 500 leave a wide margin.
const PageSize = 500

// SiteURL is where the announcements are served and the robots.txt with
// the content signals lives.
const SiteURL = "https://data.fca.org.uk/"

// HTTP is what Fetcher needs from fetch.HTTPFetcher.
type HTTP interface {
	PostJSON(ctx context.Context, url string, body []byte) ([]byte, error)
	CheckTDM(ctx context.Context, url string) (bool, error)
}

// Fetcher queries the NSM search for the newest entries.
type Fetcher struct {
	HTTP HTTP
}

func (f *Fetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	query := fmt.Sprintf(`{"from":0,"size":%d,"sort":"submitted_date","sortorder":"desc","criteriaObj":{"criteria":[],"dateCriteria":[]}}`, PageSize)
	return f.HTTP.PostJSON(ctx, url, []byte(query))
}

// CheckTDM checks the site, not the API host (which answers 403).
func (f *Fetcher) CheckTDM(ctx context.Context, _ string) (bool, error) {
	return f.HTTP.CheckTDM(ctx, SiteURL)
}
