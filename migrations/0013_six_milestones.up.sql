-- Six-milestone model: milestone_index now counts COMPLETED milestones (0-6).
-- Milestone k (1-based) is complete iff milestone_index >= k.
--   1 Meeting Completed   2 Mockup Completed   3 Design Approved
--   4 Website Completed   5 Payment Completed  6 Website is Live
--
-- Remap from the old current-stage semantics (0-7 core stages, +1 offset when
-- wants_call prepended a "Discovery Call" stage):
--   designing(0)/mockup-review(1) -> 1   approved(2)/building(3)/ready(4) -> 3
--   payment(5) -> 4   waiting-for-launch(6) -> 5   launched(7) -> 6
UPDATE leads SET milestone_index =
    CASE
        WHEN status = 'pending' THEN 0
        WHEN wants_call AND milestone_index = 0 THEN 0 -- call not yet held
        ELSE CASE milestone_index - (CASE WHEN wants_call THEN 1 ELSE 0 END)
            WHEN 0 THEN 1
            WHEN 1 THEN 1
            WHEN 2 THEN 3
            WHEN 3 THEN 3
            WHEN 4 THEN 3
            WHEN 5 THEN 4
            WHEN 6 THEN 5
            ELSE 6
        END
    END;

ALTER TABLE leads
    ADD CONSTRAINT leads_milestone_index_range CHECK (milestone_index BETWEEN 0 AND 6);
