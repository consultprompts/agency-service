-- Admin-invited leads are created without an owner and stay unattached until
-- the client redeems them (POST /agency/leads/redeem).
ALTER TABLE leads ALTER COLUMN user_id DROP NOT NULL;
