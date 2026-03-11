package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// ListingStatus represents the pipeline stage of a listing.
type ListingStatus string

const (
	StatusDraft    ListingStatus = "draft"
	StatusPrepped  ListingStatus = "prepped"
	StatusApproved ListingStatus = "approved"
	StatusPosted   ListingStatus = "posted"
	StatusFailed   ListingStatus = "failed"
	StatusArchived ListingStatus = "archived"
)

// ListingType represents the listing type (sale, rent, etc.).
type ListingType string

const (
	ListingTypeSale       ListingType = "sale"
	ListingTypeRent       ListingType = "rent"
	ListingTypeSaleOrRent ListingType = "sale_or_rent"
)

// PropertyType represents the type of property.
type PropertyType string

const (
	PropertyCondo      PropertyType = "condo"
	PropertyHouse      PropertyType = "house"
	PropertyTownhouse  PropertyType = "townhouse"
	PropertyLand       PropertyType = "land"
	PropertyCommercial PropertyType = "commercial"
	PropertyApartment  PropertyType = "apartment"
	PropertyVilla      PropertyType = "villa"
	PropertyShophouse  PropertyType = "shophouse"
	PropertyWarehouse  PropertyType = "warehouse"
	PropertyOther      PropertyType = "other"
)

// Listing represents a real estate listing in the system.
type Listing struct {
	ID         uuid.UUID `json:"id" db:"id"`
	ExternalID *string   `json:"external_id,omitempty" db:"external_id"`
	Source     string    `json:"source" db:"source"`

	// Property Details
	Title        string       `json:"title" db:"title"`
	Description  *string      `json:"description,omitempty" db:"description"`
	PropertyType PropertyType `json:"property_type" db:"property_type"`
	ListingType  ListingType  `json:"listing_type" db:"listing_type"`

	// Pricing
	Price         *float64 `json:"price,omitempty" db:"price"`
	PriceCurrency string   `json:"price_currency" db:"price_currency"`

	// Location
	Address   *string  `json:"address,omitempty" db:"address"`
	District  *string  `json:"district,omitempty" db:"district"`
	Province  *string  `json:"province,omitempty" db:"province"`
	Country   string   `json:"country" db:"country"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// Specs
	Bedrooms    *int16   `json:"bedrooms,omitempty" db:"bedrooms"`
	Bathrooms   *int16   `json:"bathrooms,omitempty" db:"bathrooms"`
	AreaSqm     *float64 `json:"area_sqm,omitempty" db:"area_sqm"`
	Floor       *int16   `json:"floor,omitempty" db:"floor"`
	TotalFloors *int16   `json:"total_floors,omitempty" db:"total_floors"`
	YearBuilt   *int16   `json:"year_built,omitempty" db:"year_built"`

	// AI / Vector
	Embedding *pgvector.Vector `json:"-" db:"embedding"`

	// Pipeline
	Status    ListingStatus `json:"status" db:"status"`
	PreppedAt *time.Time    `json:"prepped_at,omitempty" db:"prepped_at"`
	PostedAt  *time.Time    `json:"posted_at,omitempty" db:"posted_at"`

	// Metadata
	Tags      []string  `json:"tags" db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Relations (loaded separately)
	Images  []ListingImage   `json:"images,omitempty"`
	Content []ListingContent `json:"content,omitempty"`
}

// ListingImage represents an image associated with a listing.
type ListingImage struct {
	ID            uuid.UUID `json:"id" db:"id"`
	ListingID     uuid.UUID `json:"listing_id" db:"listing_id"`
	OriginalURL   *string   `json:"original_url,omitempty" db:"original_url"`
	LocalPath     *string   `json:"local_path,omitempty" db:"local_path"`
	StagedPath    *string   `json:"staged_path,omitempty" db:"staged_path"`
	IsHero        bool      `json:"is_hero" db:"is_hero"`
	AIDescription *string   `json:"ai_description,omitempty" db:"ai_description"`
	SortOrder     int16     `json:"sort_order" db:"sort_order"`
	Width         *int      `json:"width,omitempty" db:"width"`
	Height        *int      `json:"height,omitempty" db:"height"`
	FileSizeBytes *int64    `json:"file_size_bytes,omitempty" db:"file_size_bytes"`
	MimeType      *string   `json:"mime_type,omitempty" db:"mime_type"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ContentLanguage represents the supported languages.
type ContentLanguage string

const (
	LangThai    ContentLanguage = "th"
	LangEnglish ContentLanguage = "en"
	LangMyanmar ContentLanguage = "my"
)

// ListingContent represents AI-generated content for a listing.
type ListingContent struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	ListingID     uuid.UUID       `json:"listing_id" db:"listing_id"`
	Language      ContentLanguage `json:"language" db:"language"`
	Title         string          `json:"title" db:"title"`
	Body          string          `json:"body" db:"body"`
	Hashtags      []string        `json:"hashtags" db:"hashtags"`
	ModelUsed     *string         `json:"model_used,omitempty" db:"model_used"`
	PromptVersion *string         `json:"prompt_version,omitempty" db:"prompt_version"`
	GeneratedAt   time.Time       `json:"generated_at" db:"generated_at"`
}

// PostTarget represents where a listing is posted.
type PostTarget string

const (
	TargetMarketplace PostTarget = "marketplace"
	TargetPage        PostTarget = "page"
)

// PostStatus represents the status of a social media post.
type PostStatus string

const (
	PostPending PostStatus = "pending"
	PostPosting PostStatus = "posting"
	PostSuccess PostStatus = "success"
	PostFailed  PostStatus = "failed"
)

// PostHistory records each attempt to post a listing to social media.
type PostHistory struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ListingID      uuid.UUID  `json:"listing_id" db:"listing_id"`
	Target         PostTarget `json:"target" db:"target"`
	PageID         *string    `json:"page_id,omitempty" db:"page_id"`
	PageName       *string    `json:"page_name,omitempty" db:"page_name"`
	Status         PostStatus `json:"status" db:"status"`
	FBPostID       *string    `json:"fb_post_id,omitempty" db:"fb_post_id"`
	FBPostURL      *string    `json:"fb_post_url,omitempty" db:"fb_post_url"`
	ErrorMessage   *string    `json:"error_message,omitempty" db:"error_message"`
	ScreenshotPath *string    `json:"screenshot_path,omitempty" db:"screenshot_path"`
	StartedAt      *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}
