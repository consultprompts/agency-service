package repository

import (
	"context"
	"time"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MilestoneRepository struct {
	db *pgxpool.Pool
}

func NewMilestoneRepository(db *pgxpool.Pool) *MilestoneRepository {
	return &MilestoneRepository{db: db}
}

func (repo *MilestoneRepository) CreateMilestone(ctx context.Context, milestone model.Milestone) (*model.Milestone, error) {
	query := `
		INSERT INTO milestones (lead_id, title, description, sort_order, due_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at
	`

	err := repo.db.QueryRow(ctx, query,
		milestone.LeadID,
		milestone.Title,
		milestone.Description,
		milestone.SortOrder,
		milestone.DueDate,
	).Scan(&milestone.ID, &milestone.Status, &milestone.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &milestone, nil
}

func (repo *MilestoneRepository) GetMilestonesByLead(ctx context.Context, leadID string) ([]model.Milestone, error) {
	query := `
		SELECT id, lead_id, title, description, status, sort_order, due_date, completed_at, created_at
		FROM milestones
		WHERE lead_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := repo.db.Query(ctx, query, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	milestones := make([]model.Milestone, 0)
	for rows.Next() {
		var m model.Milestone
		if err := rows.Scan(
			&m.ID,
			&m.LeadID,
			&m.Title,
			&m.Description,
			&m.Status,
			&m.SortOrder,
			&m.DueDate,
			&m.CompletedAt,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		milestones = append(milestones, m)
	}

	return milestones, rows.Err()
}

// UpdateMilestone applies a partial update: nil fields keep their current
// value. completed_at is derived from status transitions.
func (repo *MilestoneRepository) UpdateMilestone(ctx context.Context, id string, title, description, status *string, sortOrder *int, dueDate *time.Time) (*model.Milestone, error) {
	query := `
		UPDATE milestones SET
			title       = COALESCE($2, title),
			description = COALESCE($3, description),
			status      = COALESCE($4, status),
			sort_order  = COALESCE($5, sort_order),
			due_date    = COALESCE($6, due_date),
			completed_at = CASE
				WHEN $4::text = 'completed' AND status <> 'completed' THEN now()
				WHEN $4::text IS NOT NULL AND $4::text <> 'completed' THEN NULL
				ELSE completed_at
			END
		WHERE id = $1
		RETURNING id, lead_id, title, description, status, sort_order, due_date, completed_at, created_at
	`

	var m model.Milestone
	err := repo.db.QueryRow(ctx, query, id, title, description, status, sortOrder, dueDate).Scan(
		&m.ID,
		&m.LeadID,
		&m.Title,
		&m.Description,
		&m.Status,
		&m.SortOrder,
		&m.DueDate,
		&m.CompletedAt,
		&m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (repo *MilestoneRepository) DeleteMilestone(ctx context.Context, id string) (bool, error) {
	tag, err := repo.db.Exec(ctx, `DELETE FROM milestones WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
