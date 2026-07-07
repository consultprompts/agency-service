ALTER TABLE leads
    ADD COLUMN mockup_url        TEXT    NULL,
    ADD COLUMN revision_feedback TEXT    NULL,
    ADD COLUMN wants_maintenance BOOLEAN NOT NULL DEFAULT false;
