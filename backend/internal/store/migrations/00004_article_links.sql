-- +goose Up
-- Current link state per (article, company): a projection of the append-only
-- article_entities, kept in the same transaction by a trigger. Reads (ListNews)
-- use this instead of DISTINCT ON over the whole history.
CREATE TABLE article_links (
    article_id bigint      NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
    company_id bigint      NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    status     link_status NOT NULL,
    decided_at timestamptz NOT NULL,
    entity_id  bigint      NOT NULL,
    PRIMARY KEY (article_id, company_id)
);
CREATE INDEX article_links_linked_company_idx ON article_links (company_id, article_id) WHERE status = 'linked';
CREATE INDEX article_links_linked_article_idx ON article_links (article_id) WHERE status = 'linked';

INSERT INTO article_links (article_id, company_id, status, decided_at, entity_id)
SELECT DISTINCT ON (article_id, company_id) article_id, company_id, status, decided_at, id
FROM article_entities
ORDER BY article_id, company_id, decided_at DESC, id DESC;

-- +goose StatementBegin
CREATE FUNCTION project_article_link() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO article_links (article_id, company_id, status, decided_at, entity_id)
    VALUES (NEW.article_id, NEW.company_id, NEW.status, NEW.decided_at, NEW.id)
    ON CONFLICT (article_id, company_id) DO UPDATE
        SET status = EXCLUDED.status, decided_at = EXCLUDED.decided_at, entity_id = EXCLUDED.entity_id
        WHERE (EXCLUDED.decided_at, EXCLUDED.entity_id) > (article_links.decided_at, article_links.entity_id);
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER article_entities_project
    AFTER INSERT ON article_entities
    FOR EACH ROW EXECUTE FUNCTION project_article_link();

-- +goose Down
DROP TRIGGER IF EXISTS article_entities_project ON article_entities;
DROP FUNCTION IF EXISTS project_article_link();
DROP TABLE IF EXISTS article_links;
