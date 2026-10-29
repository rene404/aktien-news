package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// tdmrepPaths are checked in order. The TDMRep spec puts the file under
// /.well-known/, but OTS serves it from the site root, so both count.
var tdmrepPaths = []string{"/.well-known/tdmrep.json", "/tdmrep.json"}

// maxTDMRepBytes caps a tdmrep.json response.
const maxTDMRepBytes = 1 << 20

type tdmrepEntry struct {
	Location       string `json:"location"`
	TDMReservation int    `json:"tdm-reservation"`
}

// CheckTDM reports whether the site behind sourceURL reserves text and data
// mining rights (§ 42h UrhG / Art. 4 DSM directive), via the TDM-Reservation
// header on sourceURL or a tdmrep.json entry whose location covers its path.
// A missing file (404/410) means no reservation. A reservation found
// anywhere wins; otherwise any other failure is returned as an error so the
// caller can keep the previous result.
func (f *HTTPFetcher) CheckTDM(ctx context.Context, sourceURL string) (bool, error) {
	u, err := url.Parse(sourceURL)
	if err != nil {
		return false, err
	}

	resp, err := f.do(ctx, http.MethodHead, sourceURL, "*/*")
	if err != nil {
		return false, fmt.Errorf("tdm header %s: %w", sourceURL, err)
	}
	resp.Body.Close()
	// A non-2xx HEAD (e.g. 405) carries no usable header; the files decide.
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 && strings.TrimSpace(resp.Header.Get("TDM-Reservation")) == "1" {
		return true, nil
	}

	// Both files are read even if one fails: a broken /.well-known/ must not
	// hide a reservation in the root file.
	var errs []error
	for _, p := range tdmrepPaths {
		fileURL := (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: p}).String()
		entries, found, err := f.tdmrep(ctx, fileURL)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !found {
			continue
		}
		for _, e := range entries {
			if e.TDMReservation == 1 && locationMatches(e.Location, u.EscapedPath()) {
				return true, nil
			}
		}
	}
	return false, errors.Join(errs...)
}

func (f *HTTPFetcher) tdmrep(ctx context.Context, fileURL string) ([]tdmrepEntry, bool, error) {
	resp, err := f.do(ctx, http.MethodGet, fileURL, "application/json")
	if err != nil {
		return nil, false, fmt.Errorf("tdmrep %s: %w", fileURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return nil, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, false, fmt.Errorf("tdmrep %s: HTTP %d", fileURL, resp.StatusCode)
	}
	var entries []tdmrepEntry
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxTDMRepBytes)).Decode(&entries); err != nil {
		return nil, false, fmt.Errorf("tdmrep %s: %w", fileURL, err)
	}
	return entries, true, nil
}

func (f *HTTPFetcher) do(ctx context.Context, method, url, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", accept)
	return f.client.Do(req)
}

// locationMatches applies a TDMRep location pattern to a URL path the way
// robots.txt does: a prefix match where * matches any run of characters and a
// trailing $ anchors the end. An empty location covers the whole site.
func locationMatches(pattern, path string) bool {
	if pattern == "" {
		return true
	}
	if path == "" {
		path = "/"
	}
	anchored := strings.HasSuffix(pattern, "$")
	pattern = strings.TrimSuffix(pattern, "$")

	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(path, parts[0]) {
		return false
	}
	rest := path[len(parts[0]):]
	for _, part := range parts[1:] {
		i := strings.Index(rest, part)
		if i < 0 {
			return false
		}
		rest = rest[i+len(part):]
	}
	if !anchored {
		return true
	}
	last := parts[len(parts)-1]
	return rest == "" || (len(parts) > 1 && strings.HasSuffix(path, last))
}
