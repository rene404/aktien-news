package service_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/service"
)

// The point-in-time tests anchor decisions an hour in the future (t0), so
// every article's DB-set ingested_at lies before them.

func (w *world) entity(t *testing.T, article, company int64, status string, confidence float64, at time.Time) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(), `
		INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by, decided_at)
		VALUES ($1, $2, $3, 'ticker', $4, 'engine', $5)`,
		article, company, confidence, status, at); err != nil {
		t.Fatal(err)
	}
}

func (w *world) classify(t *testing.T, article, company int64, eventType string, sentiment, confidence, novelty float64, at time.Time) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(), `
		INSERT INTO classifications (article_id, company_id, event_type, sentiment, confidence, novelty, model, prompt_version, classified_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'test', 'v0', $7)`,
		article, company, eventType, sentiment, confidence, novelty, at); err != nil {
		t.Fatal(err)
	}
}

func (w *world) ingestedAt(t *testing.T, article int64) time.Time {
	t.Helper()
	var at time.Time
	if err := w.pool.QueryRow(context.Background(), `SELECT ingested_at FROM articles WHERE id = $1`, article).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

func insightTitles(items []service.Insight) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}

func TestSearchNewsPointInTime(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	t0 := time.Now().Add(time.Hour).Truncate(time.Second)

	a1 := w.article(t, w.adhoc, 1, "Erste Q3")
	a2 := w.article(t, w.adhoc, 2, "Erste rejected later")
	a3 := w.article(t, w.adhoc, 3, "Erste pending")
	w.entity(t, a1, w.erste, "linked", 0.9, t0.Add(time.Minute))
	w.entity(t, a2, w.erste, "linked", 0.9, t0.Add(time.Minute))
	w.entity(t, a2, w.erste, "rejected", 1, t0.Add(10*time.Minute))
	w.entity(t, a3, w.erste, "pending", 0.55, t0.Add(time.Minute))

	at := func(d time.Duration) *time.Time { v := t0.Add(d); return &v }
	tests := []struct {
		name string
		asOf *time.Time
		want []string
	}{
		{"before the article was ingested", ptr(w.ingestedAt(t, a1).Add(-time.Second)), nil},
		{"before the link was decided", at(30 * time.Second), nil},
		{"after linking, before rejection", at(5 * time.Minute), []string{"Erste rejected later", "Erste Q3"}},
		{"after rejection", at(20 * time.Minute), []string{"Erste Q3"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := w.svc.SearchNews(ctx, service.SearchQuery{Ticker: "ebs", Since: w.base.Add(-time.Hour), AsOf: tc.asOf})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(insightTitles(got), tc.want) {
				t.Fatalf("titles = %v, want %v", insightTitles(got), tc.want)
			}
		})
	}
}

func TestSearchNewsItemFields(t *testing.T) {
	w := newWorld(t)
	t0 := time.Now().Add(time.Hour).Truncate(time.Second)
	a := w.article(t, w.adhoc, 1, "Erste und Post kooperieren")
	w.entity(t, a, w.erste, "linked", 0.9, t0)
	w.entity(t, a, w.post, "linked", 0.95, t0)
	asOf := t0.Add(time.Minute)

	got, err := w.svc.SearchNews(context.Background(), service.SearchQuery{Since: w.base, AsOf: &asOf})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want one per linked company", len(got))
	}
	byTicker := map[string]service.Insight{}
	for _, it := range got {
		byTicker[it.Tickers[0]] = it
	}
	ebs := byTicker["EBS"]
	if ebs.ID != a || ebs.ClusterID == 0 || ebs.URL == "" || ebs.SourceKey != "fn_adhoc" || ebs.SourceWeight != 1 ||
		ebs.LinkConfidence != 0.9 || ebs.IngestedAt.IsZero() || !ebs.PublishedAt.Equal(w.base.Add(time.Minute)) {
		t.Fatalf("EBS item = %+v", ebs)
	}
	if ebs.Classification != nil {
		t.Fatalf("unclassified item has classification %+v", ebs.Classification)
	}
	if byTicker["POST"].LinkConfidence != 0.95 {
		t.Fatalf("POST item = %+v", byTicker["POST"])
	}
}

func TestSearchNewsClassificationAsOf(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	t0 := time.Now().Add(time.Hour).Truncate(time.Second)
	a := w.article(t, w.adhoc, 1, "Erste Q3")
	w.entity(t, a, w.erste, "linked", 0.9, t0)
	w.classify(t, a, w.erste, "earnings", 0.5, 0.9, 1, t0.Add(2*time.Minute))
	w.classify(t, a, w.erste, "dividend", -0.2, 0.7, 1, t0.Add(8*time.Minute))

	search := func(d time.Duration, eventType string) []service.Insight {
		t.Helper()
		asOf := t0.Add(d)
		got, err := w.svc.SearchNews(ctx, service.SearchQuery{Since: w.base, AsOf: &asOf, EventType: eventType})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	if got := search(time.Minute, ""); len(got) != 1 || got[0].Classification != nil {
		t.Fatalf("before classification: %+v", got)
	}
	if got := search(5*time.Minute, ""); len(got) != 1 || got[0].Classification == nil ||
		got[0].Classification.EventType != "earnings" || got[0].Classification.Sentiment != 0.5 {
		t.Fatalf("first classification: %+v", got)
	}
	got := search(9*time.Minute, "")
	if len(got) != 1 || got[0].Classification == nil {
		t.Fatalf("reclassified: %+v", got)
	}
	if c := got[0].Classification; c.EventType != "dividend" || c.Sentiment != -0.2 || c.Confidence != 0.7 || c.Novelty != 1 ||
		!c.ClassifiedAt.Equal(t0.Add(8*time.Minute)) {
		t.Fatalf("newest classification = %+v", c)
	}
	if got := search(9*time.Minute, "dividend"); len(got) != 1 {
		t.Fatalf("event filter after reclassification: %d items", len(got))
	}
	if got := search(5*time.Minute, "dividend"); len(got) != 0 {
		t.Fatalf("event filter sees a future classification: %+v", got)
	}
}

func TestSearchNewsFilters(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	t0 := time.Now().Add(time.Hour).Truncate(time.Second)
	for i, title := range []string{"Erste 50% Plus", "Erste 500 Plus", "Erste Dividende"} {
		a := w.article(t, w.adhoc, i+1, title)
		w.entity(t, a, w.erste, "linked", 0.9, t0)
	}
	asOf := t0.Add(time.Minute)
	until := w.base.Add(3 * time.Minute)
	minConf := 0.95

	tests := []struct {
		name string
		q    service.SearchQuery
		want []string
	}{
		{"query is a literal substring", service.SearchQuery{Query: "50%"}, []string{"Erste 50% Plus"}},
		{"query ignores case", service.SearchQuery{Query: "dividende"}, []string{"Erste Dividende"}},
		{"until is exclusive", service.SearchQuery{Until: &until}, []string{"Erste 500 Plus", "Erste 50% Plus"}},
		{"min confidence", service.SearchQuery{MinConfidence: &minConf}, nil},
		{"limit", service.SearchQuery{Limit: 1}, []string{"Erste Dividende"}},
		{"unknown ticker is empty", service.SearchQuery{Ticker: "NOPE"}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := tc.q
			q.Since, q.AsOf = w.base, &asOf
			got, err := w.svc.SearchNews(ctx, q)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(insightTitles(got), tc.want) {
				t.Fatalf("titles = %v, want %v", insightTitles(got), tc.want)
			}
		})
	}
}

func TestSearchNewsValidation(t *testing.T) {
	w := newWorld(t)
	since := w.base
	before := since.Add(-time.Minute)
	tooHigh := 1.5
	tests := []struct {
		name string
		q    service.SearchQuery
	}{
		{"since missing", service.SearchQuery{}},
		{"until before since", service.SearchQuery{Since: since, Until: &before}},
		{"limit too high", service.SearchQuery{Since: since, Limit: 51}},
		{"min confidence above 1", service.SearchQuery{Since: since, MinConfidence: &tooHigh}},
		{"bad ticker", service.SearchQuery{Since: since, Ticker: "EBS; DROP"}},
		{"unknown event type", service.SearchQuery{Since: since, EventType: "rumour"}},
		{"query too long", service.SearchQuery{Since: since, Query: string(make([]byte, 201))}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := w.svc.SearchNews(context.Background(), tc.q)
			var ve *service.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
		})
	}
}

func TestTickerSentiment(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()
	if _, err := w.pool.Exec(ctx, `UPDATE sources SET weight = 0.5 WHERE id = $1`, w.other); err != nil {
		t.Fatal(err)
	}
	t0 := time.Now().Add(time.Hour).Truncate(time.Second)
	w.base = time.Now().Add(-2 * time.Hour).Truncate(time.Minute)

	add := func(source int64, minutes int, title, status string) int64 {
		a := w.article(t, source, minutes, title)
		w.entity(t, a, w.erste, status, 0.9, t0)
		return a
	}
	r1 := add(w.adhoc, 1, "r1", "linked")
	r2 := add(w.other, 2, "r2", "linked")
	w.cluster(t, r1, r2)
	r3 := add(w.adhoc, 3, "r3", "linked")
	add(w.adhoc, 4, "r4 unclassified", "linked")
	r5 := add(w.adhoc, 5, "r5 pending", "pending")
	old := add(w.adhoc, -8*24*60, "old", "linked")

	at := t0.Add(time.Minute)
	w.classify(t, r1, w.erste, "earnings", 0.8, 0.9, 1, at)
	w.classify(t, r2, w.erste, "earnings", 0.8, 0.9, 0.5, at)
	w.classify(t, r3, w.erste, "litigation", -0.4, 0.6, 1, at)
	w.classify(t, r5, w.erste, "litigation", -1, 1, 1, at)
	w.classify(t, old, w.erste, "litigation", -1, 1, 1, at)

	asOf := t0.Add(time.Hour)
	got, err := w.svc.TickerSentiment(ctx, service.SentimentQuery{Ticker: "EBS", Window: "7d", AsOf: &asOf})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ticker != "EBS" || got.Window != "7d" || !got.AsOf.Equal(asOf) {
		t.Fatalf("header = %+v", got)
	}
	if got.NArticles != 4 || got.NClusters != 3 || got.NClassified != 3 {
		t.Fatalf("counts = %d articles, %d clusters, %d classified; want 4, 3, 3", got.NArticles, got.NClusters, got.NClassified)
	}
	// (0.8·0.9·1·1 + 0.8·0.9·0.5·0.5 − 0.4·0.6·1·1) / (0.9 + 0.225 + 0.6)
	assertScore(t, "score_weighted", got.ScoreWeighted, 0.66/1.725)
	assertScore(t, "score_mean", got.ScoreMean, 0.4)
	if len(got.ByEventType) != 2 || got.ByEventType["earnings"].N != 2 || got.ByEventType["litigation"].N != 1 {
		t.Fatalf("by_event_type = %+v", got.ByEventType)
	}
	assertScore(t, "earnings", got.ByEventType["earnings"].Score, 0.8)
	assertScore(t, "litigation", got.ByEventType["litigation"].Score, -0.4)

	// Before the classifications existed the articles still count, the scores don't.
	before := t0.Add(30 * time.Second)
	got, err = w.svc.TickerSentiment(ctx, service.SentimentQuery{Ticker: "EBS", Window: "7d", AsOf: &before})
	if err != nil {
		t.Fatal(err)
	}
	if got.NArticles != 4 || got.NClassified != 0 || got.ScoreWeighted != nil || got.ScoreMean != nil || len(got.ByEventType) != 0 {
		t.Fatalf("before classification = %+v", got)
	}
}

func TestTickerSentimentEmptyAndErrors(t *testing.T) {
	w := newWorld(t)
	ctx := context.Background()

	got, err := w.svc.TickerSentiment(ctx, service.SentimentQuery{Ticker: "POST"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Window != "7d" || got.NArticles != 0 || got.NClusters != 0 || got.ScoreWeighted != nil || got.ScoreMean != nil {
		t.Fatalf("empty window = %+v", got)
	}

	if _, err := w.svc.TickerSentiment(ctx, service.SentimentQuery{Ticker: "NOPE"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("unknown ticker: err = %v, want ErrNotFound", err)
	}
	for _, q := range []service.SentimentQuery{{}, {Ticker: "EBS", Window: "2d"}, {Ticker: "EBS; DROP"}} {
		var ve *service.ValidationError
		if _, err := w.svc.TickerSentiment(ctx, q); !errors.As(err, &ve) {
			t.Fatalf("%+v: err = %v, want ValidationError", q, err)
		}
	}
}

func assertScore(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 1e-9 {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func ptr[T any](v T) *T { return &v }
