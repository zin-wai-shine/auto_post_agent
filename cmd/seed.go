package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
	"github.com/zinwaishine/super-agent/internal/database"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed the database with demo listings",
	Long:  `Insert realistic demo property listings for testing the dashboard and CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("db-url")
		if dbURL == "" {
			cfg, err := config.Load()
			if err == nil && cfg.Database.URL != "" {
				dbURL = cfg.Database.URL
			}
		}
		if dbURL == "" {
			return fmt.Errorf("database URL required: use --db-url")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		db, err := database.New(ctx, config.DatabaseConfig{
			URL: dbURL, MaxOpenConns: 5, MaxIdleConns: 2, ConnMaxLifetime: "5m",
		})
		if err != nil {
			return fmt.Errorf("DB connection failed: %w", err)
		}
		defer db.Close()

		fmt.Println("🌱 Seeding demo listings...")

		listings := []struct {
			title, desc, propType, listType string
			price                           float64
			addr, district, province        string
			lat, lng                        float64
			beds, baths                     int
			area                            float64
			status                          string
			tags                            []string
		}{
			{
				"Luxury Penthouse at Sukhumvit 24", "Stunning penthouse with panoramic city views, private pool, and Italian marble finishes. Located in prime Sukhumvit area.",
				"condo", "sale", 42000000, "24 Sukhumvit Rd", "Khlong Toei", "Bangkok",
				13.7226, 100.5684, 3, 3, 285.0, "posted", []string{"luxury", "penthouse", "pool", "city-view"},
			},
			{
				"Modern Condo near BTS Ari", "Fully furnished 2-bedroom condo with BTS access. Rooftop garden and co-working space available.",
				"condo", "rent", 28000, "89 Phahonyothin Rd", "Phaya Thai", "Bangkok",
				13.7795, 100.5450, 2, 1, 62.0, "prepped", []string{"bts-nearby", "furnished", "rooftop"},
			},
			{
				"Pool Villa in Hua Hin", "Private pool villa with tropical garden, 5 minutes from beach. Perfect for families or rental income.",
				"villa", "sale", 12500000, "15 Soi Hua Hin 88", "Hua Hin", "Prachuap Khiri Khan",
				12.5684, 99.9576, 4, 3, 380.0, "approved", []string{"pool", "beach-nearby", "family"},
			},
			{
				"Shophouse in Chiang Mai Old City", "Renovated heritage shophouse perfect for café or boutique hotel. Walking distance to temples.",
				"shophouse", "sale", 8900000, "42 Ratchadamnoen Rd", "Mueang", "Chiang Mai",
				18.7883, 98.9930, 3, 2, 220.0, "draft", []string{"heritage", "commercial", "old-city"},
			},
			{
				"Studio Apartment near MRT Phra Ram 9", "Compact smart-home studio. High floor, city view. Walking distance to Central Plaza and MRT.",
				"apartment", "rent", 12000, "9 Ratchadapisek Rd", "Huai Khwang", "Bangkok",
				13.7570, 100.5650, 1, 1, 28.0, "draft", []string{"studio", "smart-home", "mrt-nearby"},
			},
			{
				"Beachfront Condo in Pattaya", "Direct beach access, infinity pool, fully furnished with sea view. Tourist rental license included.",
				"condo", "sale_or_rent", 6800000, "777 Jomtien Beach Rd", "Bang Lamung", "Chonburi",
				12.8781, 100.8770, 2, 2, 78.0, "prepped", []string{"beachfront", "sea-view", "rental-license"},
			},
			{
				"Townhouse in Nonthaburi", "3-story townhouse in gated community. Close to MRT Purple Line, shopping malls, and schools.",
				"townhouse", "sale", 3200000, "88 Rattanathibet Rd", "Mueang", "Nonthaburi",
				13.8608, 100.5003, 3, 2, 145.0, "posted", []string{"gated", "mrt-nearby", "family"},
			},
			{
				"Land Plot in Phuket", "1 rai prime land plot with mountain view. Suitable for boutique resort or private villa development.",
				"land", "sale", 18000000, "99 Kamala Beach Rd", "Kathu", "Phuket",
				7.9519, 98.2818, 0, 0, 1600.0, "draft", []string{"land", "mountain-view", "investment"},
			},
			{
				"Warehouse in EEC Zone", "Modern warehouse in Eastern Economic Corridor. 3-phase power, loading dock, 24/7 security.",
				"warehouse", "rent", 180000, "Industrial Estate Rd", "Si Racha", "Chonburi",
				13.1676, 100.9321, 0, 0, 2400.0, "failed", []string{"eec", "industrial", "logistics"},
			},
			{
				"Luxury House in Nichada Thani", "Executive family home in premier international compound. Near ISB school, golf course, clubhouse.",
				"house", "rent", 150000, "1 Nichada Thani", "Pak Kret", "Nonthaburi",
				13.9111, 100.4960, 5, 4, 420.0, "approved", []string{"expat", "school-nearby", "golf"},
			},
		}

		for i, l := range listings {
			var id string
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO listings (title, description, property_type, listing_type, price,
					address, district, province, latitude, longitude,
					bedrooms, bathrooms, area_sqm, status, tags)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
				RETURNING id
			`, l.title, l.desc, l.propType, l.listType, l.price,
				l.addr, l.district, l.province, l.lat, l.lng,
				l.beds, l.baths, l.area, l.status, l.tags,
			).Scan(&id)
			if err != nil {
				fmt.Printf("  ⚠️  Failed to insert listing %d: %v\n", i+1, err)
				continue
			}

			// Add trilingual content for prepped/approved/posted listings
			if l.status == "prepped" || l.status == "approved" || l.status == "posted" {
				for _, lang := range []struct{ code, title, body string }{
					{"en", l.title, l.desc},
					{"th", "🇹🇭 " + l.title, "รายละเอียด: " + l.desc},
					{"my", "🇲🇲 " + l.title, "အသေးစိတ်: " + l.desc},
				} {
					db.Pool.Exec(ctx, `
						INSERT INTO listing_content (listing_id, language, title, body, model_used)
						VALUES ($1, $2, $3, $4, 'gpt-4o')
					`, id, lang.code, lang.title, lang.body)
				}
			}

			fmt.Printf("  ✅ %s (%s)\n", l.title, l.status)
		}

		fmt.Printf("\n🎉 Seeded %d demo listings!\n", len(listings))
		fmt.Println("   Run `super-agent serve` to see them in the dashboard.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
	seedCmd.Flags().String("db-url", "", "PostgreSQL connection URL")
}
