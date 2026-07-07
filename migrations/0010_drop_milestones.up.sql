-- The per-lead milestones table was never used: progress is tracked as a
-- single leads.milestone_index int (see 0004), matching the fixed stage list
-- shared by the frontend and lead_service.go.
DROP TABLE IF EXISTS milestones;
