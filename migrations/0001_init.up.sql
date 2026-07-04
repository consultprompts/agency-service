CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE leads (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    name        TEXT NOT NULL,
    email       TEXT NOT NULL,
    business    TEXT NOT NULL,
    message     TEXT NULL,
    package     TEXT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_leads_user_id ON leads(user_id);
CREATE INDEX idx_leads_status ON leads(status);