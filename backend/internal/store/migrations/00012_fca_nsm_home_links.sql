-- The FCA NSM terms of use allow links only to the NSM home page. Items
-- stored before linked the announcement file; canonical_url keeps that
-- address as the dedupe key.

-- +goose Up
UPDATE articles a SET url = 'https://data.fca.org.uk/#/nsm/nationalstoragemechanism'
FROM sources s
WHERE s.id = a.source_id AND s.key = 'fca_nsm';

-- +goose Down
UPDATE articles a SET url = a.canonical_url
FROM sources s
WHERE s.id = a.source_id AND s.key = 'fca_nsm';
