ALTER TABLE leads
    DROP COLUMN IF EXISTS is_paid,
    DROP COLUMN IF EXISTS paid_at,
    DROP COLUMN IF EXISTS payment_amount,
    DROP COLUMN IF EXISTS site_url;
