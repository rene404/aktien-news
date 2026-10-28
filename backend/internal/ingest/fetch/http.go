// Package fetch does the live HTTP work for ingestion, so parsers stay pure.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MaxBodyBytes caps a feed response; anything larger is treated as an error.
const MaxBodyBytes = 10 << 20

type HTTPFetcher struct {
	client    *http.Client
	userAgent string
}

func New(userAgent string, timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{client: &http.Client{Timeout: timeout}, userAgent: userAgent}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9, */*;q=0.1")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("fetch %s: read body: %w", url, err)
	}
	if len(b) > MaxBodyBytes {
		return nil, fmt.Errorf("fetch %s: body exceeds %d bytes", url, MaxBodyBytes)
	}
	return b, nil
}
