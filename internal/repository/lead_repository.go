package repository

import (
	"context"
	"os"

	"github.com/consultprompts/agency-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadRepository struct {
	db *pgxpool.Pool
	// publicURL is the gateway's externally-reachable base URL (NOT this
	// service's own port — see docker-compose: agency-service has no host
	// port mapping and is only reachable via the gateway). Used to compose
	// browser-facing logo URLs; served images are always fetched through
	// the gateway since a plain <img> tag can't attach this service's
	// internal-only auth headers.
	publicURL string
}

func NewLeadRepository(db *pgxpool.Pool) *LeadRepository {
	return &LeadRepository{db: db, publicURL: os.Getenv("AGENCY_PUBLIC_URL")}
}

// leadColumns intentionally excludes logo_data — list/detail scans only need
// logo_content_type (cheap) to know a logo exists; the actual bytes are
// fetched separately by GetLeadLogo so paginated lead queries never drag
// multi-MB blobs along for the ride.
const leadColumns = `
	id, user_id, name, email, business, message,
	existing_website, existing_website_url,
	location, site_goal, pages_needed, style_direction,
	has_logo, logo_content_type, has_brand_colors, primary_color, secondary_color,
	inspiration_urls, phone_number, contact_method, timeline,
	package, wants_call, meeting_skipped, status, pre_suspend_status, milestone_index, created_at,
	mockup_url, revision_feedback, revision_count, wants_maintenance,
	is_paid, paid_at, payment_amount, site_url, domain_renewal_date
`

func (repo *LeadRepository) scanLead(row interface {
	Scan(...any) error
}, lead *model.Lead) error {
	if err := row.Scan(
		&lead.ID,
		&lead.UserID,
		&lead.Name,
		&lead.Email,
		&lead.Business,
		&lead.Message,
		&lead.ExistingWebsite,
		&lead.ExistingWebsiteURL,
		&lead.Location,
		&lead.SiteGoal,
		&lead.PagesNeeded,
		&lead.StyleDirection,
		&lead.HasLogo,
		&lead.LogoContentType,
		&lead.HasBrandColors,
		&lead.PrimaryColor,
		&lead.SecondaryColor,
		&lead.InspirationURLs,
		&lead.PhoneNumber,
		&lead.ContactMethod,
		&lead.Timeline,
		&lead.Package,
		&lead.WantsCall,
		&lead.MeetingSkipped,
		&lead.Status,
		&lead.PreSuspendStatus,
		&lead.MilestoneIndex,
		&lead.CreatedAt,
		&lead.MockupURL,
		&lead.RevisionFeedback,
		&lead.RevisionCount,
		&lead.WantsMaintenance,
		&lead.IsPaid,
		&lead.PaidAt,
		&lead.PaymentAmount,
		&lead.SiteURL,
		&lead.DomainRenewalDate,
	); err != nil {
		return err
	}

	if lead.LogoContentType != nil && repo.publicURL != "" {
		url := repo.publicURL + "/agency/leads/" + lead.ID + "/logo"
		lead.LogoURL = &url
	}
	return nil
}

func (repo *LeadRepository) CreateLead(ctx context.Context, lead model.Lead) (*model.Lead, error) {
	query := `
		INSERT INTO leads (
			user_id, name, email, business, message,
			existing_website, existing_website_url,
			location, site_goal, pages_needed, style_direction,
			has_logo, logo_data, logo_content_type, has_brand_colors, primary_color, secondary_color,
			inspiration_urls, phone_number, contact_method, timeline,
			package, wants_call, meeting_skipped, milestone_index, status
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23, $24, $25, $26
		)
		RETURNING id, status, milestone_index, created_at
	`

	err := repo.db.QueryRow(ctx, query,
		lead.UserID,
		lead.Name,
		lead.Email,
		lead.Business,
		lead.Message,
		lead.ExistingWebsite,
		lead.ExistingWebsiteURL,
		lead.Location,
		lead.SiteGoal,
		lead.PagesNeeded,
		lead.StyleDirection,
		lead.HasLogo,
		lead.LogoData,
		lead.LogoContentType,
		lead.HasBrandColors,
		lead.PrimaryColor,
		lead.SecondaryColor,
		lead.InspirationURLs,
		lead.PhoneNumber,
		lead.ContactMethod,
		lead.Timeline,
		lead.Package,
		lead.WantsCall,
		lead.MeetingSkipped,
		lead.MilestoneIndex,
		lead.Status,
	).Scan(&lead.ID, &lead.Status, &lead.MilestoneIndex, &lead.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &lead, nil
}

func (repo *LeadRepository) GetLeadByID(ctx context.Context, id string) (*model.Lead, error) {
	query := `SELECT` + leadColumns + `FROM leads WHERE id = $1`

	var lead model.Lead
	if err := repo.scanLead(repo.db.QueryRow(ctx, query, id), &lead); err != nil {
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
		if err := repo.scanLead(rows, &lead); err != nil {
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

func (repo *LeadRepository) UpdateLead(ctx context.Context, id string, lead model.Lead) error {
	_, err := repo.db.Exec(ctx, `
		UPDATE leads SET
			name = $1, business = $2, message = $3,
			existing_website = $4, existing_website_url = $5,
			location = $6, site_goal = $7, pages_needed = $8, style_direction = $9,
			has_logo = $10, has_brand_colors = $11, primary_color = $12, secondary_color = $13,
			inspiration_urls = $14, phone_number = $15, contact_method = $16, timeline = $17,
			package = $18, wants_call = $19, meeting_skipped = $20, milestone_index = $21
		WHERE id = $22 AND status = 'pending'`,
		lead.Name, lead.Business, lead.Message,
		lead.ExistingWebsite, lead.ExistingWebsiteURL,
		lead.Location, lead.SiteGoal, lead.PagesNeeded, lead.StyleDirection,
		lead.HasLogo, lead.HasBrandColors, lead.PrimaryColor, lead.SecondaryColor,
		lead.InspirationURLs, lead.PhoneNumber, lead.ContactMethod, lead.Timeline,
		lead.Package, lead.WantsCall, lead.MeetingSkipped, lead.MilestoneIndex,
		id,
	)
	return err
}

// AttachLeadUser claims an unattached lead for userID. The user_id IS NULL
// guard makes the claim atomic: it reports false when the lead was already
// attached (or doesn't exist), so concurrent redeems can't reassign a lead.
func (repo *LeadRepository) AttachLeadUser(ctx context.Context, id, userID string) (bool, error) {
	tag, err := repo.db.Exec(ctx,
		`UPDATE leads SET user_id = $1 WHERE id = $2 AND user_id IS NULL`,
		userID, id,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (repo *LeadRepository) HasActiveLead(ctx context.Context, userID string) (bool, error) {
	var count int
	err := repo.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM leads WHERE user_id = $1 AND status NOT IN ('completed', 'launched', 'suspended')`,
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
		if err := repo.scanLead(rows, &lead); err != nil {
			return nil, err
		}
		leads = append(leads, lead)
	}

	return leads, nil
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

func (repo *LeadRepository) SetLaunched(ctx context.Context, id, siteURL string, milestoneIndex int) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET milestone_index = $1, status = 'launched', site_url = $2 WHERE id = $3`,
		milestoneIndex, siteURL, id,
	)
	return err
}

// GetLeadLogo fetches the raw logo bytes for a single lead — kept separate
// from leadColumns/scanLead so listing/detail queries never pull the blob.
// Returns pgx.ErrNoRows both when the lead doesn't exist and when it exists
// but has no logo stored, since callers (the public logo endpoint) treat
// both cases identically as 404.
func (repo *LeadRepository) GetLeadLogo(ctx context.Context, id string) ([]byte, string, error) {
	var data []byte
	var contentType *string
	err := repo.db.QueryRow(ctx,
		`SELECT logo_data, logo_content_type FROM leads WHERE id = $1`,
		id,
	).Scan(&data, &contentType)
	if err != nil {
		return nil, "", err
	}
	if data == nil || contentType == nil {
		return nil, "", pgx.ErrNoRows
	}
	return data, *contentType, nil
}

// SetLeadLogo updates a pending lead's stored logo. Called from UpdateLead
// only when the edit request actually included a new file — never as part
// of the main UpdateLead UPDATE, so editing a lead without re-uploading a
// logo can't accidentally wipe the one already on file.
func (repo *LeadRepository) SetLeadLogo(ctx context.Context, id string, data []byte, contentType string) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET logo_data = $1, logo_content_type = $2 WHERE id = $3`,
		data, contentType, id,
	)
	return err
}

func (repo *LeadRepository) SetMockupURL(ctx context.Context, id, url string) error {
	// Clear any prior revision feedback — a fresh mockup means the client's
	// last request for changes has been addressed — and leave 'revision' status.
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET mockup_url = $1, revision_feedback = NULL, status = 'accepted' WHERE id = $2`,
		url, id,
	)
	return err
}

func (repo *LeadRepository) SetRevisionFeedback(ctx context.Context, id, feedback string) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET revision_feedback = $1, status = 'revision' WHERE id = $2`,
		feedback, id,
	)
	return err
}

func (repo *LeadRepository) IncrementRevisionCount(ctx context.Context, id string) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET revision_count = revision_count + 1 WHERE id = $1`, id)
	return err
}

// SetSuspended writes the new status and the pre_suspend_status bookkeeping
// column in one shot — used both to suspend (status="suspended", preStatus =
// what it was) and to reactivate (status = the saved preStatus, preStatus =
// nil).
func (repo *LeadRepository) SetSuspended(ctx context.Context, id, status string, preStatus *string) error {
	_, err := repo.db.Exec(ctx,
		`UPDATE leads SET status = $1, pre_suspend_status = $2 WHERE id = $3`,
		status, preStatus, id,
	)
	return err
}

func (repo *LeadRepository) SetWantsMaintenance(ctx context.Context, id string, wants bool) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET wants_maintenance = $1 WHERE id = $2`, wants, id)
	return err
}

func (repo *LeadRepository) SetWantsCall(ctx context.Context, id string, wants bool) error {
	_, err := repo.db.Exec(ctx, `UPDATE leads SET wants_call = $1 WHERE id = $2`, wants, id)
	return err
}

func (repo *LeadRepository) InsertActivity(ctx context.Context, leadID, eventType string, detail *string) error {
	_, err := repo.db.Exec(ctx,
		`INSERT INTO lead_activity (lead_id, event_type, detail) VALUES ($1, $2, $3)`,
		leadID, eventType, detail,
	)
	return err
}

func (repo *LeadRepository) GetActivity(ctx context.Context, leadID string) ([]model.LeadActivity, error) {
	rows, err := repo.db.Query(ctx,
		`SELECT id, lead_id, event_type, detail, created_at FROM lead_activity WHERE lead_id = $1 ORDER BY created_at DESC`,
		leadID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activity := make([]model.LeadActivity, 0)
	for rows.Next() {
		var a model.LeadActivity
		if err := rows.Scan(&a.ID, &a.LeadID, &a.EventType, &a.Detail, &a.CreatedAt); err != nil {
			return nil, err
		}
		activity = append(activity, a)
	}
	return activity, nil
}
