package models

import (
	"encoding/json"
	"testing"
)

func TestListingStatus_Values(t *testing.T) {
	statuses := []ListingStatus{
		StatusDraft, StatusPrepped, StatusApproved,
		StatusPosted, StatusFailed, StatusArchived,
	}
	expected := []string{"draft", "prepped", "approved", "posted", "failed", "archived"}

	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("status %d: got %s, want %s", i, s, expected[i])
		}
	}
}

func TestPropertyType_Values(t *testing.T) {
	types := []PropertyType{
		PropertyCondo, PropertyHouse, PropertyTownhouse, PropertyLand,
		PropertyCommercial, PropertyApartment, PropertyVilla,
		PropertyShophouse, PropertyWarehouse, PropertyOther,
	}
	if len(types) != 10 {
		t.Errorf("expected 10 property types, got %d", len(types))
	}
}

func TestContentLanguage_Values(t *testing.T) {
	langs := []ContentLanguage{LangThai, LangEnglish, LangMyanmar}
	expected := []string{"th", "en", "my"}
	for i, l := range langs {
		if string(l) != expected[i] {
			t.Errorf("language %d: got %s, want %s", i, l, expected[i])
		}
	}
}

func TestListing_JSONSerialization(t *testing.T) {
	price := 25000.0
	beds := int16(2)
	baths := int16(1)
	area := 65.5

	listing := Listing{
		Title:        "Test Condo",
		PropertyType: PropertyCondo,
		ListingType:  ListingTypeRent,
		Price:        &price,
		Bedrooms:     &beds,
		Bathrooms:    &baths,
		AreaSqm:      &area,
		Status:       StatusDraft,
		Country:      "TH",
		Tags:         []string{"luxury", "bts-nearby"},
	}

	data, err := json.Marshal(listing)
	if err != nil {
		t.Fatalf("failed to marshal listing: %v", err)
	}

	var decoded Listing
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal listing: %v", err)
	}

	if decoded.Title != listing.Title {
		t.Errorf("title mismatch: got %s", decoded.Title)
	}
	if *decoded.Price != *listing.Price {
		t.Errorf("price mismatch: got %f", *decoded.Price)
	}
	if decoded.Status != StatusDraft {
		t.Errorf("status mismatch: got %s", decoded.Status)
	}
	if len(decoded.Tags) != 2 {
		t.Errorf("tags count mismatch: got %d", len(decoded.Tags))
	}
}

func TestListing_JSONOmitsEmpty(t *testing.T) {
	listing := Listing{
		Title:   "Minimal Listing",
		Status:  StatusDraft,
		Country: "TH",
	}

	data, err := json.Marshal(listing)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// These nullable fields should be omitted
	omittedFields := []string{"external_id", "description", "price", "bedrooms", "bathrooms"}
	for _, field := range omittedFields {
		if contains(jsonStr, `"`+field+`"`) {
			t.Errorf("field %s should be omitted when nil", field)
		}
	}
}

func TestSearchRequest_JSONRoundTrip(t *testing.T) {
	minPrice := 10000.0
	maxPrice := 50000.0
	propType := PropertyCondo

	req := SearchRequest{
		Query: "2 bed condo near BTS Ari under 30k",
		Filters: SearchFilters{
			PropertyType: &propType,
			MinPrice:     &minPrice,
			MaxPrice:     &maxPrice,
		},
		Limit:    20,
		MinScore: 0.7,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal search request: %v", err)
	}

	var decoded SearchRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal search request: %v", err)
	}

	if decoded.Query != req.Query {
		t.Errorf("query mismatch: got %s", decoded.Query)
	}
	if *decoded.Filters.MinPrice != minPrice {
		t.Errorf("min price mismatch: got %f", *decoded.Filters.MinPrice)
	}
	if *decoded.Filters.PropertyType != PropertyCondo {
		t.Errorf("property type mismatch: got %s", *decoded.Filters.PropertyType)
	}
}

func TestPipelineStats_JSON(t *testing.T) {
	stats := PipelineStats{
		TotalListings: 150,
		DraftCount:    50,
		PreppedCount:  40,
		ApprovedCount: 30,
		PostedCount:   20,
		FailedCount:   5,
		ArchivedCount: 5,
	}

	sum := stats.DraftCount + stats.PreppedCount + stats.ApprovedCount +
		stats.PostedCount + stats.FailedCount + stats.ArchivedCount
	if sum != stats.TotalListings {
		t.Errorf("stats don't add up: sum=%d, total=%d", sum, stats.TotalListings)
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if !contains(string(data), `"total_listings":150`) {
		t.Error("total_listings not found in JSON output")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
