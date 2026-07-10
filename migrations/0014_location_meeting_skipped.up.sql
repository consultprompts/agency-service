ALTER TABLE leads
    ADD COLUMN location        TEXT    NULL,
    ADD COLUMN meeting_skipped BOOLEAN NOT NULL DEFAULT false;
