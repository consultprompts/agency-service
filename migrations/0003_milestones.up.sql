CREATE TABLE milestones (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id      UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    sort_order   INT NOT NULL DEFAULT 0,
    due_date     DATE NULL,
    completed_at TIMESTAMPTZ NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_milestones_lead_id ON milestones(lead_id);
