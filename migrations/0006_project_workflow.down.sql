ALTER TABLE leads
    DROP COLUMN IF EXISTS mockup_url,
    DROP COLUMN IF EXISTS revision_feedback,
    DROP COLUMN IF EXISTS wants_maintenance;
