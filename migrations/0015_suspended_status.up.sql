-- pre_suspend_status records what the lead's status was right before it was
-- suspended, so reactivating restores it exactly (pending vs accepted vs
-- revision) instead of guessing.
ALTER TABLE leads ADD COLUMN pre_suspend_status TEXT NULL;
