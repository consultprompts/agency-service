-- Logo bytes are stored directly on the lead row and served back via
-- GET /agency/leads/:id/logo. logo_content_type also drives whether a lead
-- has a servable logo, without needing to touch the (large) logo_data column
-- from list/detail queries.
ALTER TABLE leads ADD COLUMN logo_data BYTEA;
ALTER TABLE leads ADD COLUMN logo_content_type TEXT;
