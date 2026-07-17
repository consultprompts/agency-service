-- Unattached (never-redeemed) invites cannot survive the NOT NULL constraint.
DELETE FROM leads WHERE user_id IS NULL;
ALTER TABLE leads ALTER COLUMN user_id SET NOT NULL;
