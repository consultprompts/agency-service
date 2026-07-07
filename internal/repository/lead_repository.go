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

const leadColumns = `
	id, user_id, name, email, business,
	existing_website, existing_website_url,
	site_goal, pages_needed, style_direction,
	has_logo, logo_url, has_brand_colors, primary_color, secondary_color,
	inspiration_urls, phone_number, contact_method, timeline,
	package, status, milestone_index, created_at,
	mockup_url, revision_feedback, wants_maintenance,
	is_paid, paid_at, payment_amount, site_url, domain_renewal_date
`

func scanLead(row interface {
	Scan(...any) error
}, lead *model.Lead) error {
	return row.Scan(
		&lead.ID,
		&lead.UserID,
		&lead.Name,
		&lead.Email,
		&lead.Business,
		&lead.ExistingWebsite,
		&lead.ExistingWebsiteURL,
		&lead.SiteGoal,
		&lead.PagesNeeded,
		&lead.StyleDirection,
		&lead.HasLogo,
		&lead.LogoURL,
		&lead.HasBrandColors,
		&lead.PrimaryColor,
		&lead.SecondaryColor,
		&lead.InspirationURLs,
		&lead.PhoneNumber,
		&lead.ContactMethod,
		&lead.Timeline,
		&lead.Package,
		&lead.Status,
		&lead.MilestoneIndex,
		&lead.CreatedAt,
		&lead.MockupURL,
		&lead.RevisionFeedback,
		&lead.WantsMaintenance,
		&lead.IsPaid,
		&lead.PaidAt,
		&lead.PaymentAmount,
		&lead.SiteURL,
		&lead.DomainRenewalDate,
	)
}

func (repo *LeadRepository) CreateLead(ctx context.Context, lead model.Lead) (*model.Lead, error) {
	query := `
		INSERT INTO leads (
			user_id, name, email, business,
			existing_website, existing_website_url,
			site_goal, pages_needed, style_direction,
			has_logo, logo_url, has_brand_colors, primary_color, secondary_color,
			inspiration_urls, phone_number, contact_method, timeline,
			package
		) VALUES (
			$1, $2, $3, $4,
			$5, $6,
			$7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18,
			$19
		)
		RETURNING id, status, milestone_index, created_at
	`

	err := repo.db.QueryRow(ctx, query,
		lead.UserID,
		lead.Name,
		lead.Email,
		lead.Business,
		lead.ExistingWebsite,
		lead.ExistingWebsiteURL,
		lead.SiteGoal,
		lead.PagesNeeded,
		lead.StyleDirection,
		lead.HasLogo,
		lead.LogoURL,
		lead.HasBrandColors,
		lead.PrimaryColor,
		lead.SecondaryColor,
		lead.InspirationURLs,
		lead.PhoneNumber,
		lead.ContactMethod,
		lead.Timeline,
		lead.Package,
	).Scan(&lead.ID, &lead.Status, &lead.MilestoneIndex, &lead.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeadByID(ctx context.Context, id string) (*model.Lead, error) {
	query := `SELECT` + leadColumns + `FROM leads WHERE id = $1`

	var lead model.Lead
	if err := scanLead(repo.db.QueryRow(ctx, query, id), &lead); err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeads(ctx context.Context, limit, offset int) ([]model.Lead, error) {
	query := `SELECT` + leadColumns + `FROM leads ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := repo.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leads := make([]model.Lead, 0)
	for rows.Next() {
		var lead model.Lead
		if err := scanLead(rows, &lead); err != nil {
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
		`SELECT COUNT(*) FROM leads WHERE user_id = $1 AND status NOT IN ('completed', 'launched')`,
		userID,
	).Scan(&count)
	return count > 0, err
}

func (repo *LeadRepository) GetLeadsByUserID(ctx context.Context, userID string) ([]model.Lead, error) {
	query := `SELECT` + leadColumns + `FROM leads WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := repo.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leads := make([]model.Lead, 0)
	for rows.Next() {
		var lead model.Lead
		if err := scanLead(rows, &lead); err != nil {
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
	// Never regress status from 'launched' — that can only be set by SetLaunched.
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET milestone_index = $1, status = 'accepted' WHERE id = $2 AND status != 'launched'`,
		milestoneIndex, id,
	)
	return err
}

func (repo *LeadRepository) MarkPaid(ctx context.Context, id string, amount int) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET is_paid = true, paid_at = NOW(), payment_amount = $1, domain_renewal_date = NOW() + INTERVAL '1 year' WHERE id = $2`,
		amount, id,
	)
	return err
}

func (repo *LeadRepository) SetLaunched(ctx context.Context, id, siteURL string) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET milestone_index = 5, status = 'launched', site_url = $1 WHERE id = $2`,
		siteURL, id,
	)
	return err
}

func (repo *LeadRepository) SetMockupURL(ctx context.Context, id, url string) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET mockup_url = $1 WHERE id = $2`, url, id)
	return err
}

func (repo *LeadRepository) SetRevisionFeedback(ctx context.Context, id, feedback string) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET revision_feedback = $1 WHERE id = $2`, feedback, id)
	return err
}

func (repo *LeadRepository) SetWantsMaintenance(ctx context.Context, id string, wants bool) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET wants_maintenance = $1 WHERE id = $2`, wants, id)
	return err
}
