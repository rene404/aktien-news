-- name: InsertArticle :one
-- Returns pgx.ErrNoRows when the article is already known, either by
-- (source_id, external_id) or by canonical_url.
INSERT INTO articles (
    source_id, external_id, url, canonical_url, title, lang,
    published_at, content_hash, simhash, issuer_hint, issuer_isin
) VALUES (
    @source_id, @external_id, @url, @canonical_url, @title, sqlc.narg('lang'),
    @published_at, @content_hash, @simhash, sqlc.narg('issuer_hint'), sqlc.narg('issuer_isin')
)
ON CONFLICT DO NOTHING
RETURNING *;

-- name: GetArticle :one
SELECT * FROM articles WHERE id = @id;
