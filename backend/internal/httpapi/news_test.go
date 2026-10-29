package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/httpapi"
	"github.com/rene404/aktien-news/backend/internal/service"
	"github.com/rene404/aktien-news/backend/internal/store/db"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
)

// newServer seeds the catalog testdata and one article linked to Erste (EBS).
func newServer(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("../catalog/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	src, err := q.GetSourceByKey(ctx, "fn_adhoc")
	if err != nil {
		t.Fatal(err)
	}
	a, err := q.InsertArticle(ctx, db.InsertArticleParams{
		SourceID: src.ID, ExternalID: "1", Url: "https://example.com/erste", CanonicalUrl: "https://example.com/erste",
		Title: "EQS-Adhoc: Erste Group Bank AG: Dividende", ContentHash: []byte{1},
		PublishedAt: time.Date(2026, 10, 28, 9, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		WITH c AS (INSERT INTO clusters (first_seen_at, representative_article_id) VALUES (now(), $1) RETURNING id)
		UPDATE articles SET cluster_id = (SELECT id FROM c) WHERE id = $1`, a.ID); err != nil {
		t.Fatal(err)
	}
	erste, err := catalog.New(pool).CompanyByISIN(ctx, "AT0000652011")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by)
		VALUES ($1, $2, 1, 'issuer', 'linked', 'engine')`, a.ID, erste.ID); err != nil {
		t.Fatal(err)
	}
	return httpapi.NewRouter(service.New(pool))
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestNewsEndpoint(t *testing.T) {
	h := newServer(t)
	rec := get(t, h, "/api/v1/news?ticker=EBS")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age") {
		t.Errorf("Cache-Control = %q", cc)
	}
	body := decode(t, rec)
	items := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %v", items)
	}
	item := items[0].(map[string]any)
	want := map[string]any{
		"title":        "EQS-Adhoc: Erste Group Bank AG: Dividende",
		"url":          "https://example.com/erste",
		"published_at": "2026-10-28T09:30:00Z",
	}
	for k, v := range want {
		if item[k] != v {
			t.Errorf("%s = %v, want %v", k, item[k], v)
		}
	}
	if src := item["source"].(map[string]any); src["key"] != "fn_adhoc" || src["name"] == "" {
		t.Errorf("source = %v", src)
	}
	if tickers := item["tickers"].([]any); len(tickers) != 1 || tickers[0] != "EBS" {
		t.Errorf("tickers = %v", tickers)
	}
	if body["next_cursor"] != nil {
		t.Errorf("next_cursor = %v, want null", body["next_cursor"])
	}
}

func TestNewsUnknownTickerIsEmptyArrayNotNull(t *testing.T) {
	rec := get(t, newServer(t), "/api/v1/news?ticker=NOPE")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"items":[]`) {
		t.Errorf("body = %s, want items: []", rec.Body)
	}
}

func TestNewsBadRequests(t *testing.T) {
	h := newServer(t)
	for _, target := range []string{
		"/api/v1/news?limit=0",
		"/api/v1/news?limit=101",
		"/api/v1/news?limit=abc",
		"/api/v1/news?from=yesterday",
		"/api/v1/news?from=2026-10-28T10:00:00Z&to=2026-10-28T09:00:00Z",
		"/api/v1/news?cursor=garbage",
		"/api/v1/news?ticker=%27%3B--",
	} {
		rec := get(t, h, target)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", target, rec.Code)
			continue
		}
		if msg, _ := decode(t, rec)["error"].(string); msg == "" {
			t.Errorf("%s: missing error message", target)
		}
	}
}

func TestInstrumentEndpoints(t *testing.T) {
	h := newServer(t)

	rec := get(t, h, "/api/v1/instruments?q=erst")
	if rec.Code != http.StatusOK {
		t.Fatalf("search: status = %d", rec.Code)
	}
	items := decode(t, rec)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["ticker"] != "EBS" {
		t.Errorf("search items = %v", items)
	}

	if rec := get(t, h, "/api/v1/instruments"); rec.Code != http.StatusBadRequest {
		t.Errorf("search without q: status = %d, want 400", rec.Code)
	}

	rec = get(t, h, "/api/v1/instruments/ebs")
	if rec.Code != http.StatusOK {
		t.Fatalf("detail: status = %d", rec.Code)
	}
	detail := decode(t, rec)
	if detail["name"] != "Erste Group" || detail["isin"] != "AT0000652011" || detail["mic"] != "XWBO" {
		t.Errorf("detail = %v", detail)
	}
	if idx := detail["indices"].([]any); len(idx) != 1 || idx[0] != "ATX" {
		t.Errorf("indices = %v", idx)
	}

	if rec := get(t, h, "/api/v1/instruments/NOPE"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown instrument: status = %d, want 404", rec.Code)
	}
}

func TestSourcesEndpoint(t *testing.T) {
	rec := get(t, newServer(t), "/api/v1/sources")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	items := decode(t, rec)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["key"] != "fn_adhoc" {
		t.Errorf("items = %v", items)
	}
}
