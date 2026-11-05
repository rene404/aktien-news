package service_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/service"
)

func counts(items []service.TickerCount) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = fmt.Sprintf("%s=%d", it.Ticker, it.Mentions)
	}
	return out
}

// tickerWorld: Erste in three clusters (one with two linked members, one
// from the news_api source), Post in one plus a pending and a rejected
// link, Apple in one; all at or after base.
func tickerWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	rep := w.article(t, w.adhoc, 10, "erste 1", link{w.erste, "linked"})
	member := w.article(t, w.other, 11, "erste 1 again", link{w.erste, "linked"})
	w.cluster(t, rep, member)
	w.article(t, w.other, 20, "erste 2", link{w.erste, "linked"})
	w.article(t, w.adhoc, 30, "post", link{w.post, "linked"})
	w.article(t, w.adhoc, 40, "post pending", link{w.post, "pending"})
	w.article(t, w.adhoc, 50, "post rejected", link{w.post, "linked"}, link{w.post, "rejected"})
	w.article(t, w.adhoc, 60, "apple", link{w.apple, "linked"})
	return w
}

func TestTopTickers(t *testing.T) {
	w := tickerWorld(t)
	ctx := context.Background()
	from := w.base
	cases := []struct {
		name string
		q    service.TickerQuery
		want []string
	}{
		{"all, a cluster counts once", service.TickerQuery{From: from}, []string{"EBS=2", "AAPL=1", "POST=1"}},
		{"index members only", service.TickerQuery{From: from, Index: "ATX"}, []string{"EBS=2", "POST=1"}},
		{"kinds of the representative", service.TickerQuery{From: from, Kinds: []string{"adhoc"}}, []string{"AAPL=1", "EBS=1", "POST=1"}},
		{"time of the representative", service.TickerQuery{From: from.Add(15 * time.Minute)}, []string{"AAPL=1", "EBS=1", "POST=1"}},
		{"limit", service.TickerQuery{From: from, Limit: 1}, []string{"EBS=2"}},
		{"nothing in range", service.TickerQuery{From: from.Add(2 * time.Hour)}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			items, err := w.svc.TopTickers(ctx, c.q)
			if err != nil {
				t.Fatal(err)
			}
			if got := counts(items); !slices.Equal(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
	if items, _ := w.svc.TopTickers(ctx, service.TickerQuery{From: from}); items[0].Name != "Erste Group" {
		t.Errorf("name = %q, want Erste Group", items[0].Name)
	}
}

func TestTopTickersValidation(t *testing.T) {
	w := newWorld(t)
	for _, q := range []service.TickerQuery{
		{},                                     // from missing
		{From: w.base, Limit: 21},              // limit too high
		{From: w.base, Limit: -1},              // limit too low
		{From: w.base, Index: "<x>"},           // bad index
		{From: w.base, Kinds: []string{"foo"}}, // unknown kind
	} {
		var verr *service.ValidationError
		if _, err := w.svc.TopTickers(context.Background(), q); !errors.As(err, &verr) {
			t.Errorf("%+v: err = %v, want ValidationError", q, err)
		}
	}
}

func TestIndexMembers(t *testing.T) {
	w := tickerWorld(t)
	ctx := context.Background()
	from := w.base.Add(15 * time.Minute)

	items, err := w.svc.IndexMembers(ctx, service.TickerQuery{Index: "ATX", From: from})
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by name; Apple is not in the ATX.
	if got, want := counts(items), []string{"EBS=1", "POST=1"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	all, err := w.svc.IndexMembers(ctx, service.TickerQuery{Index: "ATX"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := counts(all), []string{"EBS=2", "POST=1"}; !slices.Equal(got, want) {
		t.Fatalf("without from: got %v, want %v", got, want)
	}

	// A member without news is listed with 0; a former member is not listed.
	if _, err := w.pool.Exec(ctx, `UPDATE index_memberships SET valid_from = '2020-01-01', valid_to = '2021-01-01'
		WHERE listing_id = (SELECT id FROM listings WHERE ticker = 'EBS')`); err != nil {
		t.Fatal(err)
	}
	quiet, err := w.svc.IndexMembers(ctx, service.TickerQuery{Index: "ATX", From: w.base.Add(45 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := counts(quiet), []string{"POST=0"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestIndexMembersUnknownIndex(t *testing.T) {
	w := newWorld(t)
	if _, err := w.svc.IndexMembers(context.Background(), service.TickerQuery{Index: "NIKKEI"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	var verr *service.ValidationError
	if _, err := w.svc.IndexMembers(context.Background(), service.TickerQuery{}); !errors.As(err, &verr) {
		t.Fatalf("no index: err = %v, want ValidationError", err)
	}
}
