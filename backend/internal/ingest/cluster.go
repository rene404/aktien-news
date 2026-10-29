package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/rene404/aktien-news/backend/internal/dedupe"
	"github.com/rene404/aktien-news/backend/internal/store/db"
)

// ClusterWindow is how far apart two articles may be published and still
// report the same story.
const ClusterWindow = 72 * time.Hour

// cluster puts a freshly inserted article into the nearest cluster within
// ClusterWindow that carries the same story, or founds a new one.
func cluster(ctx context.Context, q *db.Queries, articleID int64, title string, publishedAt time.Time, hash uint64, now time.Time) error {
	cands, err := q.ClusterCandidates(ctx, db.ClusterCandidatesParams{
		ArticleID:     articleID,
		PublishedFrom: publishedAt.Add(-ClusterWindow),
		PublishedTo:   publishedAt.Add(ClusterWindow),
		Simhash:       int64(hash),
		MaxDistance:   dedupe.MaxDistance,
	})
	if err != nil {
		return fmt.Errorf("cluster candidates: %w", err)
	}
	for _, c := range cands {
		if dedupe.SameStory(title, hash, c.Title, uint64(c.Simhash)) {
			if err := q.SetArticleCluster(ctx, db.SetArticleClusterParams{ID: articleID, ClusterID: &c.ClusterID}); err != nil {
				return err
			}
			return q.RefreshCluster(ctx, c.ClusterID)
		}
	}
	id, err := q.CreateCluster(ctx, db.CreateClusterParams{FirstSeenAt: now, ArticleID: &articleID})
	if err != nil {
		return fmt.Errorf("create cluster: %w", err)
	}
	return q.SetArticleCluster(ctx, db.SetArticleClusterParams{ID: articleID, ClusterID: &id})
}

// ClusterUnassigned fingerprints and clusters articles stored before
// clustering existed, oldest first, one transaction per article. A backfilled
// cluster counts as first seen when its article was stored. An article that
// fails is logged and left for the next start; only a failure to list the
// articles is returned.
func (r *Runner) ClusterUnassigned(ctx context.Context) (failed int, err error) {
	arts, err := db.New(r.Pool).UnclusteredArticles(ctx)
	if err != nil {
		return 0, fmt.Errorf("unclustered articles: %w", err)
	}
	for _, a := range arts {
		err := pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
			q := db.New(tx)
			hash := dedupe.SimHash(a.Title)
			if a.Simhash == nil {
				if err := q.SetArticleSimHash(ctx, db.SetArticleSimHashParams{ID: a.ID, Simhash: ptr(int64(hash))}); err != nil {
					return err
				}
			} else {
				hash = uint64(*a.Simhash)
			}
			return cluster(ctx, q, a.ID, a.Title, a.PublishedAt, hash, a.IngestedAt)
		})
		if err != nil {
			if ctx.Err() != nil {
				return failed, ctx.Err()
			}
			failed++
			slog.Warn("cluster backfill failed", "article", a.ID, "err", err)
		}
	}
	return failed, nil
}

func ptr[T any](v T) *T { return &v }
