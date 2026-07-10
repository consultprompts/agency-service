ALTER TABLE leads DROP CONSTRAINT leads_milestone_index_range;

-- Best-effort reverse of the six-milestone remap back to current-stage indexes.
UPDATE leads SET milestone_index =
    CASE WHEN milestone_index = 0 THEN 0 -- meeting pending: call stage (or start) is current
    ELSE
    (CASE WHEN wants_call THEN 1 ELSE 0 END) +
    CASE milestone_index
        WHEN 1 THEN CASE WHEN mockup_url IS NULL THEN 0 ELSE 1 END
        WHEN 2 THEN 2
        WHEN 3 THEN 3
        WHEN 4 THEN 5
        WHEN 5 THEN 6
        ELSE 7
    END
    END;
