package model

import "time"

type Lead struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Business           string    `json:"business"`
	Message            *string   `json:"message,omitempty"`
	ExistingWebsite    *bool     `json:"existing_website,omitempty"`
	ExistingWebsiteURL *string   `json:"existing_website_url,omitempty"`
	Location           *string   `json:"location,omitempty"`
	SiteGoal           *string   `json:"site_goal,omitempty"`
	PagesNeeded        []string  `json:"pages_needed,omitempty"`
	StyleDirection     *string   `json:"style_direction,omitempty"`
	HasLogo            *bool     `json:"has_logo,omitempty"`
	LogoURL            *string   `json:"logo_url,omitempty"`
	HasBrandColors     *bool     `json:"has_brand_colors,omitempty"`
	PrimaryColor       *string   `json:"primary_color,omitempty"`
	SecondaryColor     *string   `json:"secondary_color,omitempty"`
	InspirationURLs    []string  `json:"inspiration_urls,omitempty"`
	PhoneNumber        *string   `json:"phone_number,omitempty"`
	ContactMethod      *string   `json:"contact_method,omitempty"`
	Timeline           *string   `json:"timeline,omitempty"`
	Package            *string   `json:"package,omitempty"`
	WantsCall          bool       `json:"wants_call"`
	MeetingSkipped     bool       `json:"meeting_skipped"`
	Status             string     `json:"status"`
	// PreSuspendStatus is internal bookkeeping only — never serialized — so a
	// suspended project can be reactivated back to whatever it was (pending,
	// accepted, revision) instead of guessing.
	PreSuspendStatus   *string    `json:"-"`
	MilestoneIndex     int        `json:"milestone_index"`
	MockupURL          *string    `json:"mockup_url,omitempty"`
	RevisionFeedback   *string    `json:"revision_feedback,omitempty"`
	RevisionCount      int        `json:"revision_count"`
	WantsMaintenance   bool       `json:"wants_maintenance"`
	IsPaid             bool       `json:"is_paid"`
	PaidAt             *time.Time `json:"paid_at,omitempty"`
	PaymentAmount      *float64   `json:"payment_amount,omitempty"`
	SiteURL            *string    `json:"site_url,omitempty"`
	DomainRenewalDate  *time.Time `json:"domain_renewal_date,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}
