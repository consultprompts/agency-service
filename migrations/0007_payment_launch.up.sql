ALTER TABLE leads
    ADD COLUMN is_paid        BOOLEAN        NOT NULL DEFAULT false,
    ADD COLUMN paid_at        TIMESTAMPTZ    NULL,
    ADD COLUMN payment_amount DECIMAL(10,2)  NULL,
    ADD COLUMN site_url       TEXT           NULL;
