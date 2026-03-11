package database

import (
	"context"
	"testing"
	"time"

	"github.com/zinwaishine/super-agent/internal/config"
)

// TestIntegration_FullPipeline tests the complete database workflow:
// connect → verify schema → CRUD listings → vector search readiness
func TestIntegration_FullPipeline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Connect
	db, err := New(ctx, config.DatabaseConfig{
		URL:             getTestDBURL(t),
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "5m",
	})
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer db.Close()

	// 2. Verify extensions are installed
	t.Run("Extensions", func(t *testing.T) {
		extensions := []string{"uuid-ossp", "vector", "pg_trgm"}
		for _, ext := range extensions {
			var installed bool
			err := db.Pool.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)", ext,
			).Scan(&installed)
			if err != nil {
				t.Fatalf("failed to check extension %s: %v", ext, err)
			}
			if !installed {
				t.Errorf("extension %s is not installed", ext)
			}
		}
	})

	// 3. Verify all tables exist
	t.Run("Tables", func(t *testing.T) {
		tables := []string{"listings", "listing_images", "listing_content", "post_history", "_migrations"}
		for _, table := range tables {
			var exists bool
			err := db.Pool.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table,
			).Scan(&exists)
			if err != nil {
				t.Fatalf("failed to check table %s: %v", table, err)
			}
			if !exists {
				t.Errorf("table %s does not exist", table)
			}
		}
	})

	// 4. Verify listings table columns
	t.Run("ListingsColumns", func(t *testing.T) {
		expectedCols := []string{
			"id", "external_id", "source", "title", "description",
			"property_type", "listing_type", "price", "price_currency",
			"address", "district", "province", "country",
			"latitude", "longitude",
			"bedrooms", "bathrooms", "area_sqm", "floor", "total_floors", "year_built",
			"embedding", "status", "prepped_at", "posted_at",
			"raw_data", "tags", "created_at", "updated_at",
		}
		for _, col := range expectedCols {
			var exists bool
			err := db.Pool.QueryRow(ctx,
				`SELECT EXISTS(
					SELECT 1 FROM information_schema.columns 
					WHERE table_name = 'listings' AND column_name = $1
				)`, col,
			).Scan(&exists)
			if err != nil {
				t.Fatalf("failed to check column %s: %v", col, err)
			}
			if !exists {
				t.Errorf("column listings.%s does not exist", col)
			}
		}
	})

	// 5. Verify enum types
	t.Run("EnumTypes", func(t *testing.T) {
		enums := map[string][]string{
			"listing_status":   {"draft", "prepped", "approved", "posted", "failed", "archived"},
			"listing_type":     {"sale", "rent", "sale_or_rent"},
			"property_type":    {"condo", "house", "townhouse", "land", "commercial", "apartment", "villa", "shophouse", "warehouse", "other"},
			"content_language": {"th", "en", "my"},
			"post_target":      {"marketplace", "page"},
			"post_status":      {"pending", "posting", "success", "failed"},
		}
		for enumName, expectedValues := range enums {
			rows, err := db.Pool.Query(ctx,
				"SELECT unnest(enum_range(NULL::"+enumName+"))::text",
			)
			if err != nil {
				t.Fatalf("failed to query enum %s: %v", enumName, err)
			}

			var values []string
			for rows.Next() {
				var v string
				if err := rows.Scan(&v); err != nil {
					t.Fatalf("failed to scan enum value: %v", err)
				}
				values = append(values, v)
			}
			rows.Close()

			if len(values) != len(expectedValues) {
				t.Errorf("enum %s: got %d values, want %d", enumName, len(values), len(expectedValues))
			}
		}
	})

	// 6. Insert a test listing
	t.Run("InsertListing", func(t *testing.T) {
		var id string
		err := db.Pool.QueryRow(ctx, `
			INSERT INTO listings (title, description, property_type, listing_type, price, 
				address, district, province, latitude, longitude, bedrooms, bathrooms, area_sqm)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING id
		`,
			"Luxury Condo near BTS Ari",
			"Beautiful 2-bedroom condo with city views, fully furnished",
			"condo", "rent", 25000.00,
			"123 Phahonyothin Rd", "Phaya Thai", "Bangkok",
			13.7795, 100.5450,
			2, 1, 65.5,
		).Scan(&id)
		if err != nil {
			t.Fatalf("failed to insert listing: %v", err)
		}
		if id == "" {
			t.Fatal("got empty ID after insert")
		}
		t.Logf("✅ Inserted listing: %s", id)

		// 7. Query it back
		t.Run("QueryBack", func(t *testing.T) {
			var title string
			var price float64
			var status string
			var bedrooms int16
			err := db.Pool.QueryRow(ctx,
				"SELECT title, price, status, bedrooms FROM listings WHERE id = $1", id,
			).Scan(&title, &price, &status, &bedrooms)
			if err != nil {
				t.Fatalf("failed to query listing: %v", err)
			}
			if title != "Luxury Condo near BTS Ari" {
				t.Errorf("title mismatch: got %s", title)
			}
			if price != 25000.00 {
				t.Errorf("price mismatch: got %f", price)
			}
			if status != "draft" {
				t.Errorf("default status should be 'draft', got %s", status)
			}
			if bedrooms != 2 {
				t.Errorf("bedrooms mismatch: got %d", bedrooms)
			}
		})

		// 8. Insert trilingual content
		t.Run("InsertContent", func(t *testing.T) {
			langs := []struct {
				code  string
				title string
				body  string
			}{
				{"th", "คอนโดหรูใกล้ BTS อารีย์", "คอนโด 2 ห้องนอนสวยงาม วิวเมือง เฟอร์นิเจอร์ครบ"},
				{"en", "Luxury Condo near BTS Ari", "Beautiful 2-bed condo with city views, fully furnished"},
				{"my", "BTS Ari အနီးရှိ ခန့်ညားသော ကွန်ဒို", "မြို့ရှုခင်းမြင်ရသော လှပသည့် အခန်း ၂ ခန်းပါ ကွန်ဒို"},
			}
			for _, lang := range langs {
				_, err := db.Pool.Exec(ctx, `
					INSERT INTO listing_content (listing_id, language, title, body, model_used)
					VALUES ($1, $2, $3, $4, $5)
				`, id, lang.code, lang.title, lang.body, "gpt-4o")
				if err != nil {
					t.Fatalf("failed to insert %s content: %v", lang.code, err)
				}
			}

			// Verify all 3 languages inserted
			var count int
			err := db.Pool.QueryRow(ctx,
				"SELECT COUNT(*) FROM listing_content WHERE listing_id = $1", id,
			).Scan(&count)
			if err != nil {
				t.Fatalf("failed to count content: %v", err)
			}
			if count != 3 {
				t.Errorf("expected 3 content rows, got %d", count)
			}
		})

		// 9. Insert images
		t.Run("InsertImages", func(t *testing.T) {
			for i, url := range []string{
				"https://example.com/condo1.jpg",
				"https://example.com/condo2.jpg",
				"https://example.com/condo3.jpg",
			} {
				isHero := i == 0
				_, err := db.Pool.Exec(ctx, `
					INSERT INTO listing_images (listing_id, original_url, is_hero, sort_order)
					VALUES ($1, $2, $3, $4)
				`, id, url, isHero, i)
				if err != nil {
					t.Fatalf("failed to insert image %d: %v", i, err)
				}
			}

			var heroCount int
			err := db.Pool.QueryRow(ctx,
				"SELECT COUNT(*) FROM listing_images WHERE listing_id = $1 AND is_hero = true", id,
			).Scan(&heroCount)
			if err != nil {
				t.Fatalf("failed to count hero images: %v", err)
			}
			if heroCount != 1 {
				t.Errorf("expected 1 hero image, got %d", heroCount)
			}
		})

		// 10. Insert post history
		t.Run("InsertPostHistory", func(t *testing.T) {
			_, err := db.Pool.Exec(ctx, `
				INSERT INTO post_history (listing_id, target, status, page_name)
				VALUES ($1, $2, $3, $4)
			`, id, "marketplace", "success", "Test Page")
			if err != nil {
				t.Fatalf("failed to insert post history: %v", err)
			}
		})

		// 11. Test status update trigger (updated_at auto-updates)
		t.Run("UpdatedAtTrigger", func(t *testing.T) {
			var beforeUpdate, afterUpdate time.Time
			err := db.Pool.QueryRow(ctx,
				"SELECT updated_at FROM listings WHERE id = $1", id,
			).Scan(&beforeUpdate)
			if err != nil {
				t.Fatalf("failed to get updated_at: %v", err)
			}

			// Small delay to ensure timestamp changes
			time.Sleep(10 * time.Millisecond)

			_, err = db.Pool.Exec(ctx,
				"UPDATE listings SET status = 'prepped', prepped_at = NOW() WHERE id = $1", id,
			)
			if err != nil {
				t.Fatalf("failed to update listing: %v", err)
			}

			err = db.Pool.QueryRow(ctx,
				"SELECT updated_at FROM listings WHERE id = $1", id,
			).Scan(&afterUpdate)
			if err != nil {
				t.Fatalf("failed to get updated_at after update: %v", err)
			}

			if !afterUpdate.After(beforeUpdate) {
				t.Errorf("updated_at trigger not working: before=%v, after=%v", beforeUpdate, afterUpdate)
			}
		})

		// 12. Test unique constraint on content language
		t.Run("UniqueContentLanguage", func(t *testing.T) {
			_, err := db.Pool.Exec(ctx, `
				INSERT INTO listing_content (listing_id, language, title, body)
				VALUES ($1, 'en', 'Duplicate', 'Should fail')
			`, id)
			if err == nil {
				t.Error("expected unique constraint violation for duplicate language, got nil")
			}
		})

		// 13. Test cascade delete
		t.Run("CascadeDelete", func(t *testing.T) {
			_, err := db.Pool.Exec(ctx, "DELETE FROM listings WHERE id = $1", id)
			if err != nil {
				t.Fatalf("failed to delete listing: %v", err)
			}

			// Verify child rows are gone
			for _, table := range []string{"listing_images", "listing_content", "post_history"} {
				var count int
				err := db.Pool.QueryRow(ctx,
					"SELECT COUNT(*) FROM "+table+" WHERE listing_id = $1", id,
				).Scan(&count)
				if err != nil {
					t.Fatalf("failed to check cascade on %s: %v", table, err)
				}
				if count != 0 {
					t.Errorf("cascade delete failed on %s: still has %d rows", table, count)
				}
			}
		})
	})

	// 14. Verify vector embedding column works
	t.Run("VectorEmbedding", func(t *testing.T) {
		// Insert a listing with a dummy embedding
		var id string
		embeddingStr := "["
		for i := 0; i < 1536; i++ {
			if i > 0 {
				embeddingStr += ","
			}
			embeddingStr += "0.001"
		}
		embeddingStr += "]"

		err := db.Pool.QueryRow(ctx, `
			INSERT INTO listings (title, embedding)
			VALUES ($1, $2::vector)
			RETURNING id
		`, "Vector Test Listing", embeddingStr).Scan(&id)
		if err != nil {
			t.Fatalf("failed to insert listing with embedding: %v", err)
		}

		// Verify we can query by vector similarity
		var title string
		var distance float64
		err = db.Pool.QueryRow(ctx, `
			SELECT title, embedding <=> $1::vector AS distance
			FROM listings
			WHERE id = $2
		`, embeddingStr, id).Scan(&title, &distance)
		if err != nil {
			t.Fatalf("failed to query vector similarity: %v", err)
		}
		if distance != 0 {
			t.Errorf("self-distance should be 0, got %f", distance)
		}
		t.Logf("✅ Vector search works! Distance to self: %f", distance)

		// Cleanup
		db.Pool.Exec(ctx, "DELETE FROM listings WHERE id = $1", id)
	})

	// 15. Verify trigram search works
	t.Run("TrigramSearch", func(t *testing.T) {
		// Insert test data
		var id string
		err := db.Pool.QueryRow(ctx, `
			INSERT INTO listings (title, description) 
			VALUES ('Penthouse Suite Sukhumvit', 'Amazing penthouse with panoramic views')
			RETURNING id
		`).Scan(&id)
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}

		// Fuzzy search using trigram similarity
		var found string
		var similarity float64
		err = db.Pool.QueryRow(ctx, `
			SELECT title, similarity(title, $1) AS sim
			FROM listings
			WHERE title % $1
			ORDER BY sim DESC
			LIMIT 1
		`, "Penthose Sukumvit").Scan(&found, &similarity) // intentional typos
		if err != nil {
			t.Fatalf("trigram search failed: %v", err)
		}
		t.Logf("✅ Trigram search found: %q (similarity: %.2f)", found, similarity)

		// Cleanup
		db.Pool.Exec(ctx, "DELETE FROM listings WHERE id = $1", id)
	})
}
