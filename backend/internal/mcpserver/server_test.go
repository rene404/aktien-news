package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/mcpserver"
	"github.com/rene404/aktien-news/backend/internal/service"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
)

const token = "s3cret"

// fixture: one fn_adhoc article "Erste Q3" linked to Erste (EBS) at t0 and
// classified as earnings at t0+1m. t0 lies in the future so the DB-set
// ingested_at is before it.
type fixture struct {
	srv       *httptest.Server
	published time.Time
	t0        time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, os.DirFS("../catalog/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	f := &fixture{
		published: time.Now().Add(-time.Hour).Truncate(time.Second).UTC(),
		t0:        time.Now().Add(time.Hour).Truncate(time.Second).UTC(),
	}
	// Separate statements: data-modifying CTEs don't see each other's rows.
	var article, company int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO articles (source_id, external_id, url, canonical_url, title, published_at, content_hash)
		SELECT id, 'x1', 'https://example.com/x1', 'https://example.com/x1', 'Erste Q3', $1, '\x01'
		FROM sources WHERE key = 'fn_adhoc' RETURNING id`, f.published).Scan(&article); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT company_id FROM listings WHERE ticker = 'EBS'`).Scan(&company); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`WITH c AS (INSERT INTO clusters (first_seen_at, representative_article_id) VALUES (now(), $1) RETURNING id)
		  UPDATE articles SET cluster_id = (SELECT id FROM c) WHERE id = $1`, []any{article}},
		{`INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by, decided_at)
		  VALUES ($1, $2, 0.9, 'ticker', 'linked', 'engine', $3)`, []any{article, company, f.t0}},
		{`INSERT INTO classifications (article_id, company_id, event_type, sentiment, confidence, novelty, model, prompt_version, classified_at)
		  VALUES ($1, $2, 'earnings', 0.6, 0.8, 1, 'test', 'v0', $3)`, []any{article, company, f.t0.Add(time.Minute)}},
	} {
		if _, err := pool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatal(err)
		}
	}
	f.srv = httptest.NewServer(mcpserver.Handler(service.New(pool), token))
	t.Cleanup(f.srv.Close)
	return f
}

// rpc posts one JSON-RPC request the way boerseonline's McpHubClient does:
// no initialize, no session, both Accept types, JSON body back.
func rpc(t *testing.T, url, auth, method string, params any) (*http.Response, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if resp.StatusCode == http.StatusOK {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("body is not JSON (%s): %s", resp.Header.Get("Content-Type"), raw)
		}
	}
	return resp, out
}

func call(t *testing.T, f *fixture, tool string, args map[string]any) map[string]any {
	t.Helper()
	resp, out := rpc(t, f.srv.URL, "Bearer "+token, "tools/call", map[string]any{"name": tool, "arguments": args})
	if resp.StatusCode != http.StatusOK || out["error"] != nil {
		t.Fatalf("tools/call %s: HTTP %d, %v", tool, resp.StatusCode, out["error"])
	}
	return out["result"].(map[string]any)
}

func TestAuth(t *testing.T) {
	h := mcpserver.Handler(service.New(nil), token)
	srv := httptest.NewServer(h)
	defer srv.Close()
	for _, auth := range []string{"", "Bearer wrong", "Basic " + token, token} {
		resp, _ := rpc(t, srv.URL, auth, "tools/list", nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("auth %q: status %d, want 401", auth, resp.StatusCode)
		}
		if resp.Header.Get("WWW-Authenticate") == "" {
			t.Errorf("auth %q: no WWW-Authenticate header", auth)
		}
	}
}

func TestToolsListAsJSON(t *testing.T) {
	f := newFixture(t)
	resp, out := rpc(t, f.srv.URL, "Bearer "+token, "tools/list", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want JSON (no SSE)", ct)
	}
	var names []string
	for _, tool := range out["result"].(map[string]any)["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"search_news", "ticker_sentiment"}) {
		t.Fatalf("tools = %v", names)
	}
}

func TestSearchNewsTool(t *testing.T) {
	f := newFixture(t)
	since := f.published.Add(-time.Hour).Format(time.RFC3339)

	res := call(t, f, "search_news", map[string]any{"ticker": "EBS", "since": since, "as_of": f.t0.Add(2 * time.Minute).Format(time.RFC3339)})
	if res["isError"] == true {
		t.Fatalf("isError: %v", res["content"])
	}
	items := res["structuredContent"].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %v", items)
	}
	it := items[0].(map[string]any)
	if it["title"] != "Erste Q3" || it["event_type"] != "earnings" || it["sentiment"] != 0.6 ||
		it["source"].(map[string]any)["key"] != "fn_adhoc" || it["tickers"].([]any)[0] != "EBS" {
		t.Fatalf("item = %v", it)
	}
	if len(res["content"].([]any)) == 0 {
		t.Fatal("no text content for clients without structured output")
	}

	// as_of reaches the service: before the link decision nothing is known.
	res = call(t, f, "search_news", map[string]any{"ticker": "EBS", "since": since, "as_of": f.t0.Add(-time.Second).Format(time.RFC3339)})
	if items := res["structuredContent"].(map[string]any)["items"].([]any); len(items) != 0 {
		t.Fatalf("as_of before link: items = %v", items)
	}

	// Linked but not yet classified: the classification fields are null.
	res = call(t, f, "search_news", map[string]any{"since": since, "as_of": f.t0.Add(30 * time.Second).Format(time.RFC3339)})
	it = res["structuredContent"].(map[string]any)["items"].([]any)[0].(map[string]any)
	if v, ok := it["event_type"]; !ok || v != nil {
		t.Fatalf("event_type = %v (present %v), want null", v, ok)
	}
}

func TestTickerSentimentTool(t *testing.T) {
	f := newFixture(t)
	res := call(t, f, "ticker_sentiment", map[string]any{"ticker": "EBS", "window": "1d", "as_of": f.t0.Add(time.Hour).Format(time.RFC3339)})
	if res["isError"] == true {
		t.Fatalf("isError: %v", res["content"])
	}
	s := res["structuredContent"].(map[string]any)
	if s["ticker"] != "EBS" || s["n_articles"] != 1.0 || s["n_clusters"] != 1.0 || s["score_weighted"] != 0.6 ||
		s["baseline"] != nil || s["by_event_type"].(map[string]any)["earnings"].(map[string]any)["n"] != 1.0 {
		t.Fatalf("sentiment = %v", s)
	}
}

func TestToolErrorsAreIsError(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		tool string
		args map[string]any
		want string
	}{
		{"ticker_sentiment", map[string]any{"ticker": "NOPE"}, "unknown ticker"},
		{"ticker_sentiment", map[string]any{"ticker": "EBS", "window": "2d"}, "window"},
		{"search_news", map[string]any{"since": "yesterday"}, "since"},
		{"search_news", map[string]any{"since": "2026-10-01T00:00:00Z", "as_of": "soon"}, "as_of"},
		{"search_news", map[string]any{"since": "2026-10-01T00:00:00Z", "event_type": "rumour"}, "event type"},
	}
	for _, tc := range tests {
		res := call(t, f, tc.tool, tc.args)
		if res["isError"] != true {
			t.Errorf("%s %v: isError = %v", tc.tool, tc.args, res["isError"])
			continue
		}
		text := res["content"].([]any)[0].(map[string]any)["text"].(string)
		if !strings.Contains(text, tc.want) {
			t.Errorf("%s %v: text %q does not mention %q", tc.tool, tc.args, text, tc.want)
		}
	}
}
