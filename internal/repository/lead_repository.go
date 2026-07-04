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

func (repo *LeadRepository) GetAllLeads(ctx context.Context) ([]model.Lead, error) {
	query := `
		SELECT id, user_id, name, email, business, message, package, status, created_at
		FROM leads
		ORDER BY created_at DESC
	`

	rows, err := repo.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leads []model.Lead
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

func (repo *LeadRepository) UpdateLeadStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE leads SET status = $1 WHERE id = $2`

	_, err := repo.db.Exec(ctx, query, status, id)
	return err
}
