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
		RETURNING id, created_at, status
	`

	err := repo.db.QueryRow(ctx, query,
		lead.UserID,
		lead.Name,
		lead.Email,
		lead.Business,
		lead.Message,
		lead.Package,
	).Scan(&lead.ID, &lead.CreatedAt, &lead.Status)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeadByID(ctx context.Context, id string) (*model.Lead, error) {
	query := `
		SELECT id, user_id, name, email, business, message, package, status, created_at
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
		&lead.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeads(ctx context.Context, limit, offset int) ([]model.Lead, error) {
	query := `
		SELECT id, user_id, name, email, business, message, package, status, created_at
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

func (repo *LeadRepository) UpdateLeadStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE leads SET status = $1 WHERE id = $2`

	_, err := repo.db.Exec(ctx, query, status, id)
	return err
}
