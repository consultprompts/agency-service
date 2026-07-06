package model

import "time"

type Lead struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	Business           string    `json:"business"`
	ExistingWebsite    *bool     `json:"existing_website,omitempty"`
	ExistingWebsiteURL *string   `json:"existing_website_url,omitempty"`
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
	Status             string    `json:"status"`
	MilestoneIndex     int       `json:"milestone_index"`
	CreatedAt          time.Time `json:"created_at"`
}
