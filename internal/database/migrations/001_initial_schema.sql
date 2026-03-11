-- ============================================================
--  Super-Agent: Initial Database Schema
--  PostgreSQL + pgvector
-- ============================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================================
--  ENUM TYPES
-- ============================================================

CREATE TYPE listing_status AS ENUM (
    'draft',        -- Just synced, not yet prepped
    'prepped',      -- Images + content generated, awaiting review
    'approved',     -- Admin approved, ready to post
    'posted',       -- Successfully posted to social
    'failed',       -- Posting failed
    'archived'      -- No longer active
);

CREATE TYPE listing_type AS ENUM (
    'sale',
    'rent',
    'sale_or_rent'
);

CREATE TYPE property_type AS ENUM (
    'condo',
    'house',
    'townhouse',
    'land',
    'commercial',
    'apartment',
    'villa',
    'shophouse',
    'warehouse',
    'other'
);

CREATE TYPE content_language AS ENUM (
    'th',   -- Thai
    'en',   -- English
    'my'    -- Myanmar
);

CREATE TYPE post_target AS ENUM (
    'marketplace',
    'page'
);

CREATE TYPE post_status AS ENUM (
    'pending',
    'posting',
    'success',
    'failed'
);

-- ============================================================
--  LISTINGS TABLE (Core)
-- ============================================================

CREATE TABLE listings (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id     TEXT,                          -- ID from the source system
    source          TEXT NOT NULL DEFAULT 'manual', -- Where this listing came from

    -- Property Details
    title           TEXT NOT NULL,
    description     TEXT,
    property_type   property_type NOT NULL DEFAULT 'condo',
    listing_type    listing_type NOT NULL DEFAULT 'rent',

    -- Pricing
    price           NUMERIC(15, 2),
    price_currency  TEXT NOT NULL DEFAULT 'THB',

    -- Location
    address         TEXT,
    district        TEXT,
    province        TEXT,
    country         TEXT NOT NULL DEFAULT 'TH',
    latitude        FLOAT8,                         -- WGS84 latitude
    longitude       FLOAT8,                         -- WGS84 longitude

    -- Specs
    bedrooms        SMALLINT,
    bathrooms       SMALLINT,
    area_sqm        NUMERIC(10, 2),
    floor           SMALLINT,
    total_floors    SMALLINT,
    year_built      SMALLINT,

    -- AI / Vector Search
    embedding       vector(1536),                  -- OpenAI text-embedding-3-small dimension

    -- Pipeline Status
    status          listing_status NOT NULL DEFAULT 'draft',
    prepped_at      TIMESTAMPTZ,
    posted_at       TIMESTAMPTZ,

    -- Metadata
    raw_data        JSONB,                         -- Original data from source (for debugging)
    tags            TEXT[] DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT uq_listings_external UNIQUE (source, external_id)
);

-- ============================================================
--  LISTING IMAGES TABLE
-- ============================================================

CREATE TABLE listing_images (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,

    -- Image Data
    original_url    TEXT,                           -- Source URL
    local_path      TEXT,                           -- Path on disk after download
    staged_path     TEXT,                           -- Path after prep (resized/watermarked)

    -- AI Analysis
    is_hero         BOOLEAN NOT NULL DEFAULT FALSE, -- Selected by Vision LLM as hero shot
    ai_description  TEXT,                           -- Vision LLM description
    sort_order      SMALLINT NOT NULL DEFAULT 0,

    -- Processing
    width           INT,
    height          INT,
    file_size_bytes BIGINT,
    mime_type       TEXT,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
--  GENERATED CONTENT TABLE (Trilingual)
-- ============================================================

CREATE TABLE listing_content (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,

    -- Content
    language        content_language NOT NULL,
    title           TEXT NOT NULL,
    body            TEXT NOT NULL,
    hashtags        TEXT[] DEFAULT '{}',

    -- Metadata
    model_used      TEXT,                          -- Which LLM model generated this
    prompt_version  TEXT,                          -- Version of the prompt template
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_content_lang UNIQUE (listing_id, language)
);

-- ============================================================
--  POST HISTORY TABLE
-- ============================================================

CREATE TABLE post_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID NOT NULL REFERENCES listings(id) ON DELETE CASCADE,

    -- Target
    target          post_target NOT NULL,
    page_id         TEXT,                          -- Facebook page ID (if target=page)
    page_name       TEXT,

    -- Result
    status          post_status NOT NULL DEFAULT 'pending',
    fb_post_id      TEXT,                          -- Facebook post ID on success
    fb_post_url     TEXT,                          -- Direct link to the post
    error_message   TEXT,
    screenshot_path TEXT,                          -- Screenshot of the posted listing

    -- Timing
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
--  INDEXES
-- ============================================================

-- Vector similarity search (HNSW — no data required to build)
CREATE INDEX idx_listings_embedding ON listings
    USING hnsw (embedding vector_cosine_ops);

-- Status-based queries (pipeline view)
CREATE INDEX idx_listings_status ON listings (status);

-- Location-based queries (lat/lng composite)
CREATE INDEX idx_listings_location ON listings (latitude, longitude)
    WHERE latitude IS NOT NULL AND longitude IS NOT NULL;

-- Price range queries
CREATE INDEX idx_listings_price ON listings (price) WHERE price IS NOT NULL;

-- Property type filtering
CREATE INDEX idx_listings_property_type ON listings (property_type);

-- Image lookup by listing
CREATE INDEX idx_images_listing ON listing_images (listing_id);

-- Content lookup by listing
CREATE INDEX idx_content_listing ON listing_content (listing_id);

-- Post history lookup
CREATE INDEX idx_posts_listing ON post_history (listing_id);
CREATE INDEX idx_posts_status ON post_history (status);

-- Source dedup
CREATE INDEX idx_listings_source ON listings (source, external_id);

-- Full-text search fallback
CREATE INDEX idx_listings_title_trgm ON listings USING GIN (title gin_trgm_ops);

-- Updated at for sync
CREATE INDEX idx_listings_updated ON listings (updated_at);

-- ============================================================
--  TRIGGER: Auto-update updated_at
-- ============================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_listings_updated_at
    BEFORE UPDATE ON listings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- pg_trgm extension enabled at top of file
