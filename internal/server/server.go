package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Server is the Super-Agent API server.
type Server struct {
	pool   *pgxpool.Pool
	host   string
	port   int
	mux    *http.ServeMux
	server *http.Server
}

//go:embed static/*
var staticFiles embed.FS

// New creates a new API server.
func New(pool *pgxpool.Pool, host string, port int) *Server {
	s := &Server{
		pool: pool,
		host: host,
		port: port,
		mux:  http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// Start begins listening for requests.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.corsMiddleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("\n  🏠 Super-Agent Command Center\n")
	fmt.Printf("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  🌐 Dashboard:  http://localhost:%d\n", s.port)
	fmt.Printf("  📡 API:        http://localhost:%d/api\n", s.port)
	fmt.Printf("  💚 Health:     http://localhost:%d/api/health\n", s.port)
	fmt.Printf("  📋 Listings:   http://localhost:%d/api/listings\n", s.port)
	fmt.Printf("  📊 Pipeline:   http://localhost:%d/api/pipeline\n", s.port)
	fmt.Printf("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  Press Ctrl+C to stop\n\n")

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) registerRoutes() {
	// API routes
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/listings", s.handleListings)
	s.mux.HandleFunc("GET /api/listings/{id}", s.handleGetListing)
	s.mux.HandleFunc("GET /api/pipeline", s.handlePipeline)
	s.mux.HandleFunc("POST /api/search", s.handleSearch)

	// Serve embedded static files (dashboard)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	s.mux.Handle("GET /", http.FileServer(http.FS(staticFS)))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

// ─── Handlers ────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "connected"
	if err := s.pool.Ping(ctx); err != nil {
		dbStatus = "disconnected"
	}

	var listingCount int
	s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM listings").Scan(&listingCount)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"version":   "0.1.0",
		"database":  dbStatus,
		"listings":  listingCount,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleListings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	status := r.URL.Query().Get("status")

	query := `
		SELECT l.id, l.title, l.description, l.property_type, l.listing_type,
			   l.price, l.price_currency, l.address, l.district, l.province,
			   l.latitude, l.longitude, l.bedrooms, l.bathrooms, l.area_sqm,
			   l.status, l.tags, l.created_at, l.updated_at
		FROM listings l
	`
	var args []interface{}
	if status != "" {
		query += " WHERE l.status = $1"
		args = append(args, status)
	}
	query += " ORDER BY l.created_at DESC LIMIT 50"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query listings")
		return
	}
	defer rows.Close()

	var listings []map[string]interface{}
	for rows.Next() {
		var (
			id, title, propType, listType, currency, st string
			desc, addr, district, province              *string
			price                                       *float64
			lat, lng                                    *float64
			beds, baths                                 *int16
			area                                        *float64
			tags                                        []string
			createdAt, updatedAt                        time.Time
		)
		err := rows.Scan(&id, &title, &desc, &propType, &listType,
			&price, &currency, &addr, &district, &province,
			&lat, &lng, &beds, &baths, &area,
			&st, &tags, &createdAt, &updatedAt)
		if err != nil {
			continue
		}

		listing := map[string]interface{}{
			"id":             id,
			"title":          title,
			"property_type":  propType,
			"listing_type":   listType,
			"price_currency": currency,
			"status":         st,
			"tags":           tags,
			"created_at":     createdAt.Format(time.RFC3339),
			"updated_at":     updatedAt.Format(time.RFC3339),
		}
		if desc != nil {
			listing["description"] = *desc
		}
		if price != nil {
			listing["price"] = *price
		}
		if addr != nil {
			listing["address"] = *addr
		}
		if district != nil {
			listing["district"] = *district
		}
		if province != nil {
			listing["province"] = *province
		}
		if lat != nil {
			listing["latitude"] = *lat
		}
		if lng != nil {
			listing["longitude"] = *lng
		}
		if beds != nil {
			listing["bedrooms"] = *beds
		}
		if baths != nil {
			listing["bathrooms"] = *baths
		}
		if area != nil {
			listing["area_sqm"] = *area
		}

		// Get image count
		var imgCount int
		s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM listing_images WHERE listing_id = $1", id).Scan(&imgCount)
		listing["image_count"] = imgCount

		// Get content languages
		contentRows, _ := s.pool.Query(ctx, "SELECT language FROM listing_content WHERE listing_id = $1", id)
		var langs []string
		for contentRows.Next() {
			var lang string
			contentRows.Scan(&lang)
			langs = append(langs, lang)
		}
		contentRows.Close()
		listing["content_languages"] = langs

		listings = append(listings, listing)
	}

	if listings == nil {
		listings = []map[string]interface{}{}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"listings": listings,
		"total":    len(listings),
	})
}

func (s *Server) handleGetListing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "listing ID required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var (
		title, propType, listType, currency, st string
		desc, addr, district, province          *string
		price                                   *float64
		lat, lng                                *float64
		beds, baths                             *int16
		area                                    *float64
		tags                                    []string
		createdAt, updatedAt                    time.Time
	)

	err := s.pool.QueryRow(ctx, `
		SELECT title, description, property_type, listing_type,
			   price, price_currency, address, district, province,
			   latitude, longitude, bedrooms, bathrooms, area_sqm,
			   status, tags, created_at, updated_at
		FROM listings WHERE id = $1
	`, id).Scan(&title, &desc, &propType, &listType,
		&price, &currency, &addr, &district, &province,
		&lat, &lng, &beds, &baths, &area,
		&st, &tags, &createdAt, &updatedAt)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "listing not found")
		return
	}

	listing := map[string]interface{}{
		"id": id, "title": title, "property_type": propType,
		"listing_type": listType, "price_currency": currency,
		"status": st, "tags": tags,
		"created_at": createdAt.Format(time.RFC3339),
		"updated_at": updatedAt.Format(time.RFC3339),
	}
	if desc != nil {
		listing["description"] = *desc
	}
	if price != nil {
		listing["price"] = *price
	}
	if addr != nil {
		listing["address"] = *addr
	}
	if district != nil {
		listing["district"] = *district
	}
	if province != nil {
		listing["province"] = *province
	}
	if lat != nil {
		listing["latitude"] = *lat
	}
	if lng != nil {
		listing["longitude"] = *lng
	}
	if beds != nil {
		listing["bedrooms"] = *beds
	}
	if baths != nil {
		listing["bathrooms"] = *baths
	}
	if area != nil {
		listing["area_sqm"] = *area
	}

	s.writeJSON(w, http.StatusOK, listing)
}

func (s *Server) handlePipeline(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats := map[string]int{
		"total": 0, "draft": 0, "prepped": 0,
		"approved": 0, "posted": 0, "failed": 0, "archived": 0,
	}

	rows, err := s.pool.Query(ctx,
		"SELECT status, COUNT(*) FROM listings GROUP BY status")
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to query pipeline")
		return
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		stats[status] = count
		total += count
	}
	stats["total"] = total

	s.writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Fuzzy text search using trigram similarity (vector search needs embeddings)
	query := `
		SELECT id, title, description, property_type, listing_type, 
			   price, status, district, province, bedrooms, bathrooms, area_sqm,
			   similarity(title, $1) AS sim
		FROM listings
		WHERE title % $1 OR description % $1
		ORDER BY sim DESC
		LIMIT $2
	`
	rows, err := s.pool.Query(ctx, query, req.Query, req.Limit)
	if err != nil {
		// Fallback to ILIKE if trigram doesn't match
		query = `
			SELECT id, title, description, property_type, listing_type,
				   price, status, district, province, bedrooms, bathrooms, area_sqm,
				   0.0 AS sim
			FROM listings
			WHERE title ILIKE '%' || $1 || '%' OR description ILIKE '%' || $1 || '%'
			ORDER BY created_at DESC
			LIMIT $2
		`
		rows, err = s.pool.Query(ctx, query, req.Query, req.Limit)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "search failed")
			return
		}
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var (
			id, title, propType, listType, st string
			desc, district, province          *string
			price                             *float64
			beds, baths                       *int16
			area                              *float64
			sim                               float64
		)
		err := rows.Scan(&id, &title, &desc, &propType, &listType,
			&price, &st, &district, &province, &beds, &baths, &area, &sim)
		if err != nil {
			continue
		}

		result := map[string]interface{}{
			"id": id, "title": title, "property_type": propType,
			"listing_type": listType, "status": st,
			"similarity_score": sim,
		}
		if desc != nil {
			result["description"] = *desc
		}
		if price != nil {
			result["price"] = *price
		}
		if district != nil {
			result["district"] = *district
		}
		if province != nil {
			result["province"] = *province
		}
		if beds != nil {
			result["bedrooms"] = *beds
		}
		if baths != nil {
			result["bathrooms"] = *baths
		}
		if area != nil {
			result["area_sqm"] = *area
		}

		results = append(results, result)
	}

	if results == nil {
		results = []map[string]interface{}{}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   len(results),
		"query":   req.Query,
	})
}

// helper
func contains(s string, sub string) bool {
	return strings.Contains(s, sub)
}
