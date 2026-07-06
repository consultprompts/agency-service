package repository

import (
	"context"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadRepository struct {
	db *pgxpool.Pool
}

func NewLeadRepository(db *pgxpool.Pool) *LeadRepository {
	return &LeadRepository{db: db}
}

func (repo *LeadRepository) CreateLead(ctx context.Context, lead model.Lead) (*model.Lead, error) {
	query := `
		INSERT INTO leads (user_id, name, email, business, message, package)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, status, milestone_index
	`

	err := repo.db.QueryRow(ctx, query,
		lead.UserID,
		lead.Name,
		lead.Email,
		lead.Business,
		lead.Message,
		lead.Package,
	).Scan(&lead.ID, &lead.CreatedAt, &lead.Status, &lead.MilestoneIndex)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeadByID(ctx context.Context, id string) (*model.Lead, error) {
	query := `
		SELECT id, user_id, name, email, business, message, package, status, milestone_index, created_at
		FROM leads
		WHERE id = $1
	`

	var lead model.Lead
	err := repo.db.QueryRow(ctx, query, id).Scan(
		&lead.ID,
		&lead.UserID,
		&lead.Name,
		&lead.Email,
		&lead.Business,
		&lead.Message,
		&lead.Package,
		&lead.Status,
		&lead.MilestoneIndex,
		&lead.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeads(ctx context.Context, limit, offset int) ([]model.Lead, error) {
	query := `
		SELECT id, user_id, name, email, business, message, package, status, milestone_index, created_at
		FROM leads
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := repo.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leads := make([]model.Lead, 0)
	for rows.Next() {
		var lead model.Lead
		if err := rows.Scan(
			&lead.ID,
			&lead.UserID,
			&lead.Name,
			&lead.Email,
			&lead.Business,
			&lead.Message,
			&lead.Package,
			&lead.Status,
			&lead.MilestoneIndex,
			&lead.CreatedAt,
		); err != nil {
			return nil, err
		}
		leads = append(leads, lead)
	}

	return leads, nil
}

func (repo *LeadRepository) CountLeads(ctx context.Context) (int, error) {
	var total int
	err := repo.db.QueryRow(ctx, `SELECT COUNT(*) FROM leads`).Scan(&total)
	return total, err
}

func (repo *LeadRepository) HasActiveLead(ctx context.Context, userID string) (bool, error) {
	var count int
	err := repo.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM leads WHERE user_id = $1 AND status != 'completed'`,
		userID,
	).Scan(&count)
	return count > 0, err
}

func (repo *LeadRepository) GetLeadsByUserID(ctx context.Context, userID string) ([]model.Lead, error) {
	query := `
		SELECT id, user_id, name, email, business, message, package, status, milestone_index, created_at
		FROM leads
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := repo.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leads := make([]model.Lead, 0)
	for rows.Next() {
		var lead model.Lead
		if err := rows.Scan(
			&lead.ID,
			&lead.UserID,
			&lead.Name,
			&lead.Email,
			&lead.Business,
			&lead.Message,
			&lead.Package,
			&lead.Status,
			&lead.MilestoneIndex,
			&lead.CreatedAt,
		); err != nil {
			return nil, err
		}
		leads = append(leads, lead)
	}

	return leads, nil
}

func (repo *LeadRepository) UpdateLeadStatus(ctx context.Context, id string, status string) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (repo *LeadRepository) UpdateLeadMilestone(ctx context.Context, id string, milestoneIndex int) error {
	const milestoneCount = 5
	status := "accepted"
	if milestoneIndex >= milestoneCount-1 {
		status = "completed"
	}
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET milestone_index = $1, status = $2 WHERE id = $3`,
		milestoneIndex, status, id,
	)
	return err
}
