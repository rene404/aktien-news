package ingest_test

import (
	"context"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/store/db"
)

func TestStoreTextFollowsEffectivePolicy(t *testing.T) {
	r, pool, _, c := newRunner(t)
	ctx := context.Background()
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	q := db.New(pool)
	var articleID int64
	if err := pool.QueryRow(ctx, `SELECT min(id) FROM articles`).Scan(&articleID); err != nil {
		t.Fatal(err)
	}
	const retention = 30 * 24 * time.Hour

	cases := []struct {
		name     string
		policy   db.TextPolicy
		reserved bool
		want     bool
	}{
		{"none", db.TextPolicyNone, false, false},
		{"feed fields only", db.TextPolicyFeedFields, false, false},
		{"full text under a TDM reservation", db.TextPolicyFullText, true, false},
		{"full text", db.TextPolicyFullText, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, `DELETE FROM article_texts`); err != nil {
				t.Fatal(err)
			}
			src := db.Source{Key: "x", TextPolicy: tc.policy, TdmReserved: tc.reserved}
			stored, err := ingest.StoreText(ctx, q, src, articleID, "Body text.", c.now, retention)
			if err != nil {
				t.Fatal(err)
			}
			if stored != tc.want || count(t, pool, `SELECT count(*) FROM article_texts`) != map[bool]int{true: 1}[tc.want] {
				t.Fatalf("stored = %v, rows = %d, want stored %v", stored, count(t, pool, `SELECT count(*) FROM article_texts`), tc.want)
			}
			if !tc.want {
				return
			}
			var purgeAfter time.Time
			if err := pool.QueryRow(ctx, `SELECT purge_after FROM article_texts`).Scan(&purgeAfter); err != nil {
				t.Fatal(err)
			}
			if want := c.now.Add(retention); !purgeAfter.Equal(want) {
				t.Errorf("purge_after = %v, want %v", purgeAfter, want)
			}
		})
	}
}

func TestPurgeTextsDeletesOnlyExpired(t *testing.T) {
	r, pool, _, c := newRunner(t)
	ctx := context.Background()
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	articles := count(t, pool, `SELECT count(*) FROM articles`)
	links := count(t, pool, `SELECT count(*) FROM article_entities`)
	if _, err := pool.Exec(ctx, `
		INSERT INTO article_texts (article_id, body, purge_after)
		SELECT id, 'body', CASE WHEN id % 2 = 0 THEN $1::timestamptz - interval '1 second' ELSE $1::timestamptz + interval '1 hour' END
		FROM articles`, c.now); err != nil {
		t.Fatal(err)
	}
	expired := count(t, pool, `SELECT count(*) FROM articles WHERE id % 2 = 0`)

	n, err := ingest.PurgeTexts(ctx, pool, c.now)
	if err != nil {
		t.Fatal(err)
	}
	if int(n) != expired {
		t.Errorf("purged %d, want %d", n, expired)
	}
	if left := count(t, pool, `SELECT count(*) FROM article_texts WHERE article_id % 2 = 0`); left != 0 {
		t.Errorf("%d expired texts left", left)
	}
	if got := count(t, pool, `SELECT count(*) FROM article_texts`); got != articles-expired {
		t.Errorf("%d texts left, want %d", got, articles-expired)
	}
	if count(t, pool, `SELECT count(*) FROM articles`) != articles || count(t, pool, `SELECT count(*) FROM article_entities`) != links {
		t.Error("purge touched articles or links")
	}
}
