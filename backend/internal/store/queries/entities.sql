-- name: LatestEntities :many
-- Current link state per company = the newest row (append-only table).
SELECT DISTINCT ON (company_id)
    company_id, status, confidence::float8 AS confidence, method, decided_at
FROM article_entities
WHERE article_id = @article_id
ORDER BY company_id, decided_at DESC, id DESC;

-- name: InsertEntity :exec
INSERT INTO article_entities (article_id, company_id, confidence, method, status, decided_by)
VALUES (@article_id, @company_id, sqlc.arg('confidence')::float8, @method, @status, 'engine');

-- name: LockArticle :exec
SELECT pg_advisory_xact_lock(@article_id::bigint);

-- name: IndexCompanies :many
SELECT id, name, legal_name, cik, curated FROM companies ORDER BY id;

-- name: IndexAliases :many
SELECT company_id, alias, kind FROM company_aliases ORDER BY company_id, id;

-- name: IndexListings :many
SELECT company_id, ticker, mic, isin FROM listings WHERE active ORDER BY company_id, id;
