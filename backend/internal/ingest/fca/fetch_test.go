package fca_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rene404/aktien-news/backend/internal/ingest/fca"
)

type fakeHTTP struct {
	url, tdmURL string
	body        []byte
}

func (f *fakeHTTP) PostJSON(_ context.Context, url string, body []byte) ([]byte, error) {
	f.url, f.body = url, body
	return []byte(`{"hits":{"hits":[]}}`), nil
}

func (f *fakeHTTP) CheckTDM(_ context.Context, url string) (bool, error) {
	f.tdmURL = url
	return true, nil
}

// The search takes the newest entries first as a JSON POST.
func TestFetchPostsTheNewestFirst(t *testing.T) {
	h := &fakeHTTP{}
	const api = "https://api.data.fca.org.uk/search?index=nsm-search"
	if _, err := (&fca.Fetcher{HTTP: h}).Fetch(context.Background(), api); err != nil {
		t.Fatal(err)
	}
	var q struct {
		From      int    `json:"from"`
		Size      int    `json:"size"`
		Sort      string `json:"sort"`
		SortOrder string `json:"sortorder"`
	}
	if err := json.Unmarshal(h.body, &q); err != nil {
		t.Fatal(err)
	}
	if h.url != api || q.From != 0 || q.Size != fca.PageSize || q.Sort != "submitted_date" || q.SortOrder != "desc" {
		t.Errorf("POST %s %s", h.url, h.body)
	}
}

// The API host answers robots.txt with 403; the content signals are on the
// site that serves the announcements.
func TestCheckTDMAsksTheSite(t *testing.T) {
	h := &fakeHTTP{}
	reserved, err := (&fca.Fetcher{HTTP: h}).CheckTDM(context.Background(), "https://api.data.fca.org.uk/search?index=nsm-search")
	if err != nil || !reserved || h.tdmURL != "https://data.fca.org.uk/" {
		t.Errorf("reserved=%v err=%v url=%q", reserved, err, h.tdmURL)
	}
}
