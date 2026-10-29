-- +goose Up
-- A source that is the issuer's own channel (IR feed): every article it
-- publishes is about this company.
ALTER TABLE sources ADD COLUMN issuer_company_id bigint REFERENCES companies (id);

-- +goose Down
ALTER TABLE sources DROP COLUMN issuer_company_id;
