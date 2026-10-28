// Package dedupe recognises the same article across fetches and sources.
package dedupe

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Query parameters that only track the click and never identify content.
var trackingParams = map[string]bool{
	"fbclid": true, "gclid": true, "dclid": true, "msclkid": true,
	"mc_cid": true, "mc_eid": true, "yclid": true, "_hsenc": true, "_hsmi": true,
}

// CanonicalURL returns the dedupe key for a link: https, lowercase host
// without "www." and default port, no fragment, no tracking parameters,
// remaining parameters sorted, no trailing slash except for the root.
// The original link is stored separately for display.
func CanonicalURL(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("canonical url %q: %w", raw, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("canonical url %q: unsupported scheme", raw)
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", errors.New("canonical url: missing host")
	}
	host = strings.TrimPrefix(host, "www.")
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		host += ":" + p
	}

	q := u.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "utm_") || trackingParams[lk] {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var query []string
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			query = append(query, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
	}

	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}

	out := "https://" + host + path
	if len(query) > 0 {
		out += "?" + strings.Join(query, "&")
	}
	return out, nil
}

// TitleHash is a sha256 over the case- and whitespace-normalised title.
func TitleHash(title string) []byte {
	norm := strings.Join(strings.Fields(strings.ToLower(title)), " ")
	sum := sha256.Sum256([]byte(norm))
	return sum[:]
}
