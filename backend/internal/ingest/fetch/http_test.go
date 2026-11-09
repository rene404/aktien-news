package fetch_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest/fetch"
)

func TestFetchSendsUserAgentAndReturnsBody(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte("<rss/>"))
	}))
	defer srv.Close()

	f := fetch.New("aktien-news-bot/1.0 (+https://aktien-news.at)", 5*time.Second)
	b, err := f.Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "<rss/>" {
		t.Errorf("body = %q", b)
	}
	if ua != "aktien-news-bot/1.0 (+https://aktien-news.at)" {
		t.Errorf("user agent = %q", ua)
	}
}

func TestFetchNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := fetch.New("ua", 5*time.Second).Fetch(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v, want an error mentioning 403", err)
	}
}

func TestFetchLimitsBodySize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", fetch.MaxBodyBytes+1)))
	}))
	defer srv.Close()

	if _, err := fetch.New("ua", 5*time.Second).Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("oversized body: want error")
	}
}

// DILA archives reach 17 MB; their fetcher raises the cap, the default stays.
func TestFetchWithMaxBytesRaisesTheCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", fetch.MaxBodyBytes+1)))
	}))
	defer srv.Close()

	b, err := fetch.New("ua", 5*time.Second).WithMaxBytes(fetch.MaxBodyBytes+10).Fetch(context.Background(), srv.URL)
	if err != nil || len(b) != fetch.MaxBodyBytes+1 {
		t.Fatalf("len = %d, err = %v", len(b), err)
	}
	if _, err := fetch.New("ua", 5*time.Second).WithMaxBytes(100).Fetch(context.Background(), srv.URL); err == nil {
		t.Fatal("body above the lower cap: want error")
	}
}

// Search APIs such as the FCA NSM take the query as a JSON POST.
func TestPostJSONSendsBodyAndReturnsResponse(t *testing.T) {
	var method, ctype, ua, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, ctype, ua = r.Method, r.Header.Get("Content-Type"), r.Header.Get("User-Agent")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{"hits":{}}`))
	}))
	defer srv.Close()

	got, err := fetch.New("ua", 5*time.Second).PostJSON(context.Background(), srv.URL, []byte(`{"size":5}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"hits":{}}` || method != http.MethodPost || ctype != "application/json" || ua != "ua" || body != `{"size":5}` {
		t.Errorf("got %q; request %s %q ua=%q body=%q", got, method, ctype, ua, body)
	}
}

func TestPostJSONNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	if _, err := fetch.New("ua", 5*time.Second).PostJSON(context.Background(), srv.URL, nil); err == nil {
		t.Fatal("want error")
	}
}
