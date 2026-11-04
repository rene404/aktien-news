package httpapi_test

import (
	"net/http"
	"testing"
)

func TestTopTickersEndpoint(t *testing.T) {
	h := newServer(t)

	rec := get(t, h, "/api/v1/tickers/top?from=2026-10-28T00:00:00Z&index=ATX&kind=adhoc,ir&limit=5")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=60" {
		t.Errorf("Cache-Control = %q", cc)
	}
	items := decode(t, rec)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items = %v", items)
	}
	first := items[0].(map[string]any)
	if first["ticker"] != "EBS" || first["name"] != "Erste Group" || first["count"] != 1.0 {
		t.Errorf("item = %v", first)
	}

	// After the only article: an empty array, not null.
	rec = get(t, h, "/api/v1/tickers/top?from=2026-10-29T00:00:00Z")
	if items, ok := decode(t, rec)["items"].([]any); !ok || len(items) != 0 {
		t.Errorf("empty: items = %v", decode(t, rec)["items"])
	}

	for _, target := range []string{
		"/api/v1/tickers/top",                                      // from missing
		"/api/v1/tickers/top?from=gestern",                         // bad from
		"/api/v1/tickers/top?from=2026-10-28T00:00:00Z&limit=x",    // bad limit
		"/api/v1/tickers/top?from=2026-10-28T00:00:00Z&limit=99",   // limit too high
		"/api/v1/tickers/top?from=2026-10-28T00:00:00Z&kind=other", // unknown kind
	} {
		if rec := get(t, h, target); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", target, rec.Code)
		}
	}
}

func TestIndexInstrumentsEndpoint(t *testing.T) {
	h := newServer(t)

	rec := get(t, h, "/api/v1/indices/ATX/instruments?from=2026-10-28T00:00:00Z")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=60" {
		t.Errorf("Cache-Control = %q", cc)
	}
	items := decode(t, rec)["items"].([]any)
	got := map[string]any{}
	for _, it := range items {
		m := it.(map[string]any)
		got[m["ticker"].(string)] = m["count"]
	}
	if len(got) != 2 || got["EBS"] != 1.0 || got["POST"] != 0.0 {
		t.Errorf("counts = %v, want EBS=1 POST=0", got)
	}

	if rec := get(t, h, "/api/v1/indices/ATX/instruments"); rec.Code != http.StatusOK {
		t.Errorf("without from: status = %d", rec.Code)
	}
	if rec := get(t, h, "/api/v1/indices/NIKKEI/instruments"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown index: status = %d, want 404", rec.Code)
	}
	if rec := get(t, h, "/api/v1/indices/ATX/instruments?kind=nope"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad kind: status = %d, want 400", rec.Code)
	}
}
