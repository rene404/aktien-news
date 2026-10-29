-- name: ClusterCandidates :many
-- First articles of clusters near the given fingerprint and publication time,
-- closest first. Comparing with the first article only keeps a cluster within
-- one window instead of chaining look-alikes day after day. The caller makes
-- the final same-story decision.
SELECT a.cluster_id::bigint AS cluster_id, a.title, a.simhash::bigint AS simhash
FROM articles a
WHERE a.cluster_id IS NOT NULL
  AND a.simhash IS NOT NULL
  AND a.id <> @article_id
  AND NOT EXISTS (
      SELECT 1 FROM articles o
      WHERE o.cluster_id = a.cluster_id AND o.id <> @article_id
        AND (o.published_at, o.id) < (a.published_at, a.id))
  AND a.published_at BETWEEN @published_from::timestamptz AND @published_to::timestamptz
  AND bit_count((a.simhash # @simhash::bigint)::bit(64)) <= @max_distance::int
ORDER BY bit_count((a.simhash # @simhash::bigint)::bit(64)), a.published_at, a.id;

-- name: CreateCluster :one
INSERT INTO clusters (first_seen_at, representative_article_id)
VALUES (@first_seen_at, @article_id)
RETURNING id;

-- name: SetArticleCluster :exec
UPDATE articles SET cluster_id = @cluster_id WHERE id = @id;

-- name: RefreshCluster :exec
-- Size and representative follow from the members: the source with the
-- highest weight wins, ties go to the earliest article. first_seen_at is the
-- earliest time a member was stored (matters for backfilled clusters).
UPDATE clusters c
SET size = (SELECT count(*) FROM articles a WHERE a.cluster_id = c.id),
    first_seen_at = LEAST(c.first_seen_at, (SELECT min(a.ingested_at) FROM articles a WHERE a.cluster_id = c.id)),
    representative_article_id = (
        SELECT a.id FROM articles a JOIN sources s ON s.id = a.source_id
        WHERE a.cluster_id = c.id
        ORDER BY s.weight DESC, a.published_at, a.id
        LIMIT 1)
WHERE c.id = @id;

-- name: UnclusteredArticles :many
SELECT id, title, published_at, ingested_at, simhash FROM articles
WHERE cluster_id IS NULL
ORDER BY published_at, id;

-- name: SetArticleSimHash :exec
UPDATE articles SET simhash = @simhash WHERE id = @id;
