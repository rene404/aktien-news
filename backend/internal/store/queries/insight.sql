-- Point-in-time queries for the internal MCP tools. Every row counts only what
-- was known at @as_of: articles by ingested_at, links by the newest
-- article_entities row with decided_at <= @as_of (not the article_links
-- projection, which is "now"), classifications by classified_at <= @as_of.

-- name: SearchInsights :many
-- One row per (article, company) linked at @as_of, newest first.
WITH links AS (
    SELECT DISTINCT ON (e.article_id, e.company_id)
           e.article_id, e.company_id, e.status, e.confidence
    FROM article_entities e
    JOIN articles a ON a.id = e.article_id
    WHERE e.decided_at <= @as_of::timestamptz
      AND a.ingested_at <= @as_of::timestamptz
      AND a.published_at >= @since::timestamptz
      AND (sqlc.narg('until')::timestamptz IS NULL OR a.published_at < sqlc.narg('until')::timestamptz)
    ORDER BY e.article_id, e.company_id, e.decided_at DESC, e.id DESC
), cls AS (
    SELECT DISTINCT ON (c.article_id, c.company_id)
           c.article_id, c.company_id, c.event_type::text AS event_type, c.sentiment::float8 AS sentiment,
           c.confidence::float8 AS confidence, c.novelty::float8 AS novelty, c.classified_at
    FROM classifications c
    JOIN links k ON k.article_id = c.article_id AND k.company_id = c.company_id
    WHERE c.classified_at <= @as_of::timestamptz
    ORDER BY c.article_id, c.company_id, c.classified_at DESC, c.id DESC
)
SELECT a.id, a.cluster_id, a.title, a.url, a.published_at, a.ingested_at,
       s.key AS source_key, s.name AS source_name, s.weight::float8 AS source_weight,
       co.name AS company_name,
       coalesce((SELECT array_agg(l.ticker ORDER BY l.ticker) FROM listings l
                 WHERE l.company_id = co.id AND l.active), '{}')::text[] AS tickers,
       k.confidence::float8 AS link_confidence,
       cls.event_type,
       cls.sentiment,
       cls.confidence,
       cls.novelty,
       cls.classified_at
FROM links k
JOIN articles a   ON a.id = k.article_id
JOIN sources s    ON s.id = a.source_id
JOIN companies co ON co.id = k.company_id
LEFT JOIN cls ON cls.article_id = k.article_id AND cls.company_id = k.company_id
WHERE k.status = 'linked'
  AND k.confidence >= sqlc.arg('min_confidence')::float8
  AND (sqlc.narg('ticker')::text IS NULL OR EXISTS (
        SELECT 1 FROM listings l
        WHERE l.company_id = k.company_id AND l.active AND upper(l.ticker) = upper(sqlc.narg('ticker')::text)))
  AND (sqlc.narg('title_pattern')::text IS NULL OR a.title ILIKE sqlc.narg('title_pattern')::text)
  AND (sqlc.narg('event_type')::text IS NULL OR cls.event_type = sqlc.narg('event_type')::text)
ORDER BY a.published_at DESC, a.id DESC, co.name
LIMIT sqlc.arg('lim')::int;

-- name: SentimentRows :many
-- Articles linked to @company_id at @as_of and published in (from_ts, as_of],
-- with their newest classification at @as_of if any.
WITH links AS (
    SELECT DISTINCT ON (e.article_id) e.article_id, e.status
    FROM article_entities e
    JOIN articles a ON a.id = e.article_id
    WHERE e.company_id = @company_id
      AND e.decided_at <= @as_of::timestamptz
      AND a.ingested_at <= @as_of::timestamptz
      AND a.published_at > @from_ts::timestamptz
      AND a.published_at <= @as_of::timestamptz
    ORDER BY e.article_id, e.decided_at DESC, e.id DESC
), cls AS (
    SELECT DISTINCT ON (c.article_id)
           c.article_id, c.event_type::text AS event_type, c.sentiment::float8 AS sentiment,
           c.confidence::float8 AS confidence, c.novelty::float8 AS novelty
    FROM classifications c
    JOIN links k ON k.article_id = c.article_id
    WHERE c.company_id = @company_id AND c.classified_at <= @as_of::timestamptz
    ORDER BY c.article_id, c.classified_at DESC, c.id DESC
)
SELECT a.id, a.cluster_id, s.weight::float8 AS source_weight,
       cls.event_type,
       cls.sentiment,
       cls.confidence,
       cls.novelty
FROM links k
JOIN articles a ON a.id = k.article_id
JOIN sources s  ON s.id = a.source_id
LEFT JOIN cls ON cls.article_id = k.article_id
WHERE k.status = 'linked'
ORDER BY a.id;

-- name: CompanyForTicker :one
SELECT c.id, c.name
FROM listings l JOIN companies c ON c.id = l.company_id
WHERE l.active AND upper(l.ticker) = upper(sqlc.arg('ticker')::text)
ORDER BY l.mic
LIMIT 1;
