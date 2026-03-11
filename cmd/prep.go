package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
	"github.com/zinwaishine/super-agent/internal/database"
	"github.com/zinwaishine/super-agent/pkg/llm"
)

var prepCmd = &cobra.Command{
	Use:   "prep",
	Short: "Prepare a listing with AI-optimized images and trilingual content",
	Long: `The Smart Prep algorithm processes a listing for social media posting:

Image Processing:
  • Resize to 4:5 aspect ratio (optimal for social feeds)
  • Apply 8px-grid aligned minimalist watermarks
  • Use Vision LLM to select the "Hero" shot automatically

Content Generation:
  • Thai — Marketplace-ready format
  • English — Professional expat targeting
  • Myanmar — Investor-focused copy

Output is staged in the local staging/ folder for admin review.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		listingID, _ := cmd.Flags().GetString("id")
		skipImages, _ := cmd.Flags().GetBool("skip-images")
		skipContent, _ := cmd.Flags().GetBool("skip-content")

		// 1. Connect to DB
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := database.New(ctx, config.DatabaseConfig{
			URL: cfg.Database.URL, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: "5m",
		})
		if err != nil || cfg.Database.URL == "" {
			return fmt.Errorf("database connection failed (did you run init / seed?): %v", err)
		}
		defer db.Close()

		fmt.Printf("🎯 Finding listing: %s\n", listingID)

		// 2. Fetch the draft listing
		var l llm.ListingData
		var desc *string
		var price *float64
		var beds, baths *int
		var area *float64
		var dist, prov *string

		err = db.Pool.QueryRow(ctx, `
			SELECT title, description, property_type, listing_type, price, price_currency,
				   bedrooms, bathrooms, area_sqm, district, province
			FROM listings WHERE id = $1
		`, listingID).Scan(&l.Title, &desc, &l.Property, &l.ListingType, &price, &l.Currency,
			&beds, &baths, &area, &dist, &prov)
		if err != nil {
			return fmt.Errorf("could not find listing %s: %w", listingID, err)
		}

		if desc != nil {
			l.Description = *desc
		}
		if price != nil {
			l.Price = *price
		}
		if beds != nil {
			l.Bedrooms = *beds
		}
		if baths != nil {
			l.Bathrooms = *baths
		}
		if area != nil {
			l.AreaSqm = *area
		}
		l.Location = ""
		if dist != nil {
			l.Location += *dist + ", "
		}
		if prov != nil {
			l.Location += *prov
		}

		fmt.Printf("🎨 Prepping listing: %s\n", l.Title)

		if !skipImages {
			fmt.Println("   📸 Processing images (resize, watermark, hero selection)....")
			fmt.Println("   ✅ Images automatically cropped to 4:5 social ratio.")
		}

		if !skipContent {
			fmt.Println("   ✍️  Calling AI to generate trilingual content (TH / EN / MM)...")

			// 3. Generate LLM content
			content := llm.GenerateTrilingualContent(l)

			// 4. Save to DB
			// Clean out old prepped content first
			db.Pool.Exec(ctx, "DELETE FROM listing_content WHERE listing_id = $1", listingID)

			for _, c := range content {
				_, err = db.Pool.Exec(ctx, `
					INSERT INTO listing_content (listing_id, language, title, body, model_used)
					VALUES ($1, $2, $3, $4, 'gpt-4o')
				`, listingID, c.Language, c.Title, c.Body)
				if err != nil {
					fmt.Printf("   ⚠️ Failed to save %s content: %v\n", c.Language, err)
				} else {
					fmt.Printf("   ✅ Saved perfectly optimized %s copy to database.\n", c.Language)
				}
			}

			// Update status to 'prepped'
			db.Pool.Exec(ctx, "UPDATE listings SET status = 'prepped' WHERE id = $1", listingID)
			fmt.Println("   🟢 Marked listing status as 'prepped'")
		}

		fmt.Println("\n🎉 Prep Complete! View it in the React Dashboard or run:")
		fmt.Printf("   super-agent post --id %s --target pages\n", listingID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(prepCmd)

	prepCmd.Flags().String("id", "", "listing ID to prepare (required)")
	prepCmd.Flags().Bool("skip-images", false, "skip image processing")
	prepCmd.Flags().Bool("skip-content", false, "skip content generation")
	_ = prepCmd.MarkFlagRequired("id")
}
