# 🏠 Super-Agent CLI

> **One command to prep images, generate trilingual content, and post to multiple Facebook Pages/Marketplace.**

Super-Agent is a high-performance, open-source tool that lets any real estate admin connect their existing website database and automate the entire **"Listing-to-Social"** pipeline.

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🔄 **Universal Sync** | Connect any database, API, or CSV to import listings |
| 🧠 **Vector Search** | pgvector-powered natural language search ("Condo near BTS Ari under 20k") |
| 🎨 **Smart Prep** | AI-powered image resize, watermark, and hero-shot selection |
| ✍️ **Trilingual Content** | Auto-generate Thai, English, and Myanmar listing copy |
| 📤 **Sniper Post** | Automated Facebook Marketplace & Pages posting via Playwright |
| 🖥️ **React Dashboard** | Premium Command Center UI with Spotlight Search |

---

## 🚀 Quick Start

### Prerequisites

- **Go** 1.21+
- **Docker** & Docker Compose
- **Node.js** 20+ (for the React UI, coming soon)

### 1. Clone & Setup

```bash
git clone https://github.com/zinwaishine/super-agent.git
cd super-agent
```

### 2. Start the Database

```bash
# Spins up PostgreSQL 17 with pgvector extension
make db-up
```

### 3. Build & Initialize

```bash
# Install dependencies and build
make build

# Initialize config with your API keys (BYOK)
./build/super-agent init \
  --db-url "postgres://superagent:superagent_dev_2024@localhost:5432/superagent?sslmode=disable" \
  --openai-key "sk-your-key-here"

# Run database migrations
make migrate
```

### 4. Verify

```bash
./build/super-agent --help
```

---

## 🏗️ Project Structure

```
super-agent/
├── main.go                          # Entrypoint
├── cmd/                             # CLI commands (Cobra)
│   ├── root.go                      # Root command + global flags
│   ├── init.go                      # Config initialization
│   ├── sync.go                      # Universal data connector
│   ├── serve.go                     # API server for React UI
│   ├── prep.go                      # Smart Prep algorithm
│   ├── post.go                      # Sniper automation (Playwright)
│   └── migrate.go                   # Database migrations
├── internal/                        # Private application code
│   ├── config/                      # YAML config management
│   │   └── config.go
│   ├── database/                    # PostgreSQL + pgvector
│   │   ├── database.go              # Connection pool
│   │   ├── migrate.go               # Migration runner
│   │   └── migrations/
│   │       └── 001_initial_schema.sql
│   └── models/                      # Domain models
│       ├── listing.go               # Listing, Image, Content, PostHistory
│       └── search.go                # Search request/response
├── pkg/                             # Public reusable packages
│   └── embedding/                   # Vector embedding providers
│       └── embedding.go             # OpenAI + Ollama providers
├── ui/                              # React 19 Dashboard (Phase 4)
├── staging/                         # Prepped content for review
├── docker-compose.yml               # PostgreSQL + pgvector
├── Makefile                         # Developer workflow
└── README.md
```

---

## 🗄️ Database Schema

The schema uses **PostgreSQL** with three key extensions:

| Extension | Purpose |
|-----------|---------|
| `pgvector` | Vector similarity search on listing embeddings |
| `PostGIS` | Geospatial queries (location-based search) |
| `pg_trgm` | Fuzzy text search fallback |

### Core Tables

- **`listings`** — Property data + vector embedding + pipeline status
- **`listing_images`** — Images with AI hero-shot selection
- **`listing_content`** — Trilingual AI-generated content (TH/EN/MY)
- **`post_history`** — Social media posting audit trail

---

## 🔑 BYOK (Bring Your Own Key)

Super-Agent never stores keys on any server. All API keys are stored locally in `~/.super-agent.yaml`:

```yaml
database:
  url: postgres://user:pass@localhost:5432/superagent
llm:
  openai_key: sk-your-key
  claude_key: sk-ant-your-key
  ollama_url: http://localhost:11434
  default_model: gpt-4o
  embed_model: text-embedding-3-small
```

---

## 📋 CLI Commands

| Command | Description |
|---------|-------------|
| `super-agent init` | Setup configuration (DB, API keys) |
| `super-agent migrate` | Run database migrations |
| `super-agent sync` | Import listings + generate embeddings |
| `super-agent serve` | Start API server for React dashboard |
| `super-agent prep --id <ID>` | AI image prep + content generation |
| `super-agent post --id <ID>` | Post to Facebook Marketplace/Pages |

---

## 🛠️ Development

```bash
# Full dev setup (DB + deps + build)
make dev

# Individual commands
make build       # Build binary
make test        # Run tests
make lint        # Run linter
make db-up       # Start PostgreSQL
make db-down     # Stop PostgreSQL
make db-reset    # Reset database
make migrate     # Run migrations
```

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.
