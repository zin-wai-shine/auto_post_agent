package models

// SearchRequest represents a natural language search query from the frontend.
type SearchRequest struct {
	Query    string        `json:"query"`               // Natural language query
	Filters  SearchFilters `json:"filters,omitempty"`   // Optional structured filters
	Limit    int           `json:"limit,omitempty"`     // Max results (default 20)
	Offset   int           `json:"offset,omitempty"`    // Pagination offset
	MinScore float64       `json:"min_score,omitempty"` // Minimum similarity score (0-1)
}

// SearchFilters provides structured filters for narrowing search results.
type SearchFilters struct {
	PropertyType *PropertyType  `json:"property_type,omitempty"`
	ListingType  *ListingType   `json:"listing_type,omitempty"`
	MinPrice     *float64       `json:"min_price,omitempty"`
	MaxPrice     *float64       `json:"max_price,omitempty"`
	MinBedrooms  *int16         `json:"min_bedrooms,omitempty"`
	MaxBedrooms  *int16         `json:"max_bedrooms,omitempty"`
	MinBathrooms *int16         `json:"min_bathrooms,omitempty"`
	MinAreaSqm   *float64       `json:"min_area_sqm,omitempty"`
	MaxAreaSqm   *float64       `json:"max_area_sqm,omitempty"`
	Province     *string        `json:"province,omitempty"`
	District     *string        `json:"district,omitempty"`
	NearLat      *float64       `json:"near_lat,omitempty"`  // Center lat for radius search
	NearLng      *float64       `json:"near_lng,omitempty"`  // Center lng for radius search
	RadiusKm     *float64       `json:"radius_km,omitempty"` // Search radius in km
	Status       *ListingStatus `json:"status,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
}

// SearchResult wraps a listing with its similarity score.
type SearchResult struct {
	Listing         Listing `json:"listing"`
	SimilarityScore float64 `json:"similarity_score"` // Cosine similarity (0-1)
	Distance        float64 `json:"distance"`         // Vector distance (lower = more similar)
}

// SearchResponse is the API response for search queries.
type SearchResponse struct {
	Results     []SearchResult `json:"results"`
	Total       int            `json:"total"`
	Query       string         `json:"query"`
	TimeTakenMs int64          `json:"time_taken_ms"`
}

// PipelineStats represents the overview stats for the dashboard.
type PipelineStats struct {
	TotalListings int `json:"total_listings"`
	DraftCount    int `json:"draft_count"`
	PreppedCount  int `json:"prepped_count"`
	ApprovedCount int `json:"approved_count"`
	PostedCount   int `json:"posted_count"`
	FailedCount   int `json:"failed_count"`
	ArchivedCount int `json:"archived_count"`
}
