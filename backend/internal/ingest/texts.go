package ingest

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

// StoreText keeps an article's body for the classifier, but only when the
// source's effective policy is full_text (configured so, and no TDM
// reservation). The text is purged after retention. It reports whether
// anything was stored.
func StoreText(ctx context.Context, q *db.Queries, src db.Source, articleID int64, body string, now time.Time, retention time.Duration) (bool, error) {
	if EffectivePolicy(src.TextPolicy, src.TdmReserved) != db.TextPolicyFullText {
		return false, nil
	}
	err := q.UpsertArticleText(ctx, db.UpsertArticleTextParams{
		ArticleID: articleID, Body: body, FetchedAt: now, PurgeAfter: now.Add(retention),
	})
	return err == nil, err
}

// PurgeTexts deletes texts whose retention has run out. Articles and their
// links stay.
func PurgeTexts(ctx context.Context, pool *pgxpool.Pool, now time.Time) (int64, error) {
	return db.New(pool).PurgeExpiredTexts(ctx, now)
}
