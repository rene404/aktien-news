package fetch_test

import (
	"context"
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
