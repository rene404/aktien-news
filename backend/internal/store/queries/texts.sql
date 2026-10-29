-- name: UpsertArticleText :exec
-- Internal only; never served by any API.
INSERT INTO article_texts (article_id, body, fetched_at, purge_after)
VALUES (@article_id, @body, @fetched_at, @purge_after)
ON CONFLICT (article_id) DO UPDATE
SET body = EXCLUDED.body, fetched_at = EXCLUDED.fetched_at, purge_after = EXCLUDED.purge_after;

-- name: PurgeExpiredTexts :execrows
DELETE FROM article_texts WHERE purge_after <= @now;
