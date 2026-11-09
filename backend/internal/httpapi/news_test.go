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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/httpapi"
	"github.com/rene404/aktien-news/backend/internal/service"
	"github.com/rene404/aktien-news/backend/internal/store/db"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
)

// newServer seeds the catalog testdata and one article linked to Erste (EBS).
func newServer(t *testing.T) http.Handler {
	t.Helper()
	h, _ := newServerWithPool(t)
	return h
}

func newServerWithPool(t *testing.T) (http.Handler, *pgxpool.Pool) {
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
	return httpapi.NewRouter(service.New(pool)), pool
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
	if src := item["source"].(map[string]any); src["key"] != "fn_adhoc" || src["name"] == "" || src["kind"] != "adhoc" {
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
		"/api/v1/news?kind=adhoc,gossip",
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
	if len(items) != 1 || items[0].(map[string]any)["key"] != "fn_adhoc" || items[0].(map[string]any)["kind"] != "adhoc" {
		t.Errorf("items = %v", items)
	}
	// No run yet: present, but null.
	if v, ok := items[0].(map[string]any)["last_ok_at"]; !ok || v != nil {
		t.Errorf("last_ok_at = %v (present %v), want null", v, ok)
	}
}

func TestNewsAlsoAt(t *testing.T) {
	h, pool := newServerWithPool(t)
	ctx := context.Background()
	// A copy of the Erste ad-hoc in another source, in the same cluster.
	if _, err := pool.Exec(ctx, `
		INSERT INTO sources (key, name, kind, market, url, parser) VALUES ('ots_finanzen', 'APA-OTS Finanzen', 'press_release', 'AT', 'https://example.com/ots', 'ots');
		INSERT INTO articles (source_id, external_id, url, canonical_url, title, published_at, content_hash, cluster_id)
		SELECT s.id, 'o1', 'https://example.com/ots/1', 'https://example.com/ots/1', 'Erste Group: Dividende', '2026-10-28T09:35:00Z', '\x02', a.cluster_id
		FROM sources s, articles a WHERE s.key = 'ots_finanzen' AND a.external_id = '1';`); err != nil {
		t.Fatal(err)
	}
	items := decode(t, get(t, h, "/api/v1/news"))["items"].([]any)
	also := items[0].(map[string]any)["also"].([]any)
	if len(also) != 1 {
		t.Fatalf("also = %v", also)
	}
	a := also[0].(map[string]any)
	src := a["source"].(map[string]any)
	if a["title"] != "Erste Group: Dividende" || a["url"] != "https://example.com/ots/1" || src["name"] != "APA-OTS Finanzen" || a["published_at"] != "2026-10-28T09:35:00Z" {
		t.Errorf("also[0] = %v", a)
	}
}

func TestNewsAlsoAtEmptyIsArray(t *testing.T) {
	items := decode(t, get(t, newServer(t), "/api/v1/news"))["items"].([]any)
	if also, ok := items[0].(map[string]any)["also"].([]any); !ok || len(also) != 0 {
		t.Errorf("also = %v, want []", items[0].(map[string]any)["also"])
	}
}

// same_source: further items of the shown source in the same cluster.
func TestNewsSameSource(t *testing.T) {
	h, pool := newServerWithPool(t)
	if v := decode(t, get(t, h, "/api/v1/news"))["items"].([]any)[0].(map[string]any)["same_source"]; v != float64(0) {
		t.Errorf("alone: same_source = %v, want 0", v)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO articles (source_id, external_id, url, canonical_url, title, published_at, content_hash, cluster_id)
		SELECT a.source_id, '1b', 'https://example.com/1b', 'https://example.com/1b', a.title, a.published_at + interval '1 minute', '\x03', a.cluster_id
		FROM articles a WHERE a.external_id = '1';`); err != nil {
		t.Fatal(err)
	}
	if v := decode(t, get(t, h, "/api/v1/news"))["items"].([]any)[0].(map[string]any)["same_source"]; v != float64(1) {
		t.Errorf("same_source = %v, want 1", v)
	}
}

func TestSourcesEndpointLastOK(t *testing.T) {
	h, pool := newServerWithPool(t)
	if _, err := pool.Exec(context.Background(), `INSERT INTO ingest_runs (source_id, started_at, finished_at, status)
		SELECT id, '2026-11-05T19:55:00Z', '2026-11-05T19:55:30Z', 'ok' FROM sources WHERE key = 'fn_adhoc'`); err != nil {
		t.Fatal(err)
	}
	items := decode(t, get(t, h, "/api/v1/sources"))["items"].([]any)
	if got := items[0].(map[string]any)["last_ok_at"]; got != "2026-11-05T19:55:30Z" {
		t.Errorf("last_ok_at = %v", got)
	}
}

func TestNewsKindFilter(t *testing.T) {
	h := newServer(t)
	for target, want := range map[string]int{
		"/api/v1/news?kind=adhoc":                1,
		"/api/v1/news?kind=filing,press_release": 0,
		"/api/v1/news?kind=":                     1, // empty means no filter
	} {
		rec := get(t, h, target)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d", target, rec.Code)
			continue
		}
		if items := decode(t, rec)["items"].([]any); len(items) != want {
			t.Errorf("%s: %d items, want %d", target, len(items), want)
		}
	}
}
