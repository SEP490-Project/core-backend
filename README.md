# SEP490 Core Backend

Core backend service for the BShowSell platform — an e-commerce and social media management system. Built with Go 1.24 using clean architecture, supporting brand management, product catalog, order processing, content publishing, social media integration, and analytics.

## Tech Stack

- **Language**: Go 1.24.0
- **Database**: PostgreSQL + TimescaleDB (time-series analytics)
- **Message Broker**: RabbitMQ (async processing)
- **Cache/Queue**: Redis/Valkey (caching + Asynq task scheduling)
- **Object Storage**: AWS S3 + CloudFront
- **Payments**: PayOS
- **Shipping**: GHN (Giao Hang Nhanh)
- **Social**: Facebook Graph API, TikTok Open API
- **AI**: Google Gemini, OpenRouter, Moonshot
- **Notifications**: Firebase Cloud Messaging (push), Gmail SMTP (email)
- **Observability**: OpenTelemetry (traces, logs, metrics)
- **Auth**: JWT RS256 with optional HashiCorp Vault integration

## Project Structure

```
cmd/
  server/              Main API server entry point
  crypto_cli/          CLI tool for RSA key pair generation
config/                Configuration files and templates
  config.yaml.template Main config template
  .env.template        Environment variables template
  rabbitmq_config.yaml RabbitMQ topology definition
internal/
  domain/              Models, enums, constants, state machines
  application/         Services, DTOs, interfaces
  infrastructure/      Repositories, proxies, message consumers, cron jobs
  presentation/        HTTP handlers, middleware, router, WebSocket
pkg/
  crypto/              RSA key generation, token encryption
  logging/             Zap + OpenTelemetry logging bridge
  file/                Image/video processing, FFmpeg utilities
  utils/               General-purpose helpers
  tiptap/              Tiptap rich text parser
  goroutine/           Worker pool implementation
migrations/            SQL migration files (chronologically ordered)
tests/
  unit_tests/          Isolated function tests
  integration_tests/   End-to-end workflow tests
  performance_tests/   Benchmarks and load tests
  fixtures/            Reusable test data factories
templates/             Email notification templates
docs/                  Generated Swagger documentation
```

## Features

| Module            | Description                                                           |
| ----------------- | --------------------------------------------------------------------- |
| **Auth**          | JWT RS256, refresh tokens, social login (Facebook, TikTok)            |
| **Users**         | Role-based access (Admin, Brand, Staff, KOC)                          |
| **Products**      | Catalog, variants, options, categories, reviews, Excel import         |
| **Orders**        | Full lifecycle with shipping address, GHN integration, pre-orders     |
| **Contracts**     | Advertising, affiliate, brand ambassador, co-producing                |
| **Campaigns**     | Campaign management with milestones and budget tracking               |
| **Payments**      | PayOS integration, deposit tracking, refund handling                  |
| **Content**       | Blog posts, content publishing, scheduling, social media posting      |
| **Social**        | Facebook page management, TikTok video publishing, metrics polling    |
| **Analytics**     | Sales, marketing, content engagement, brand partner, CTR tracking     |
| **AI**            | Multi-provider AI chat (Gemini, OpenRouter, Moonshot)                 |
| **Notifications** | Email, push (FCM), in-app via RabbitMQ consumers                      |
| **Real-time**     | WebSocket and SSE for live updates                                    |
| **Scheduling**    | Asynq-based task scheduler (content publishing, payment expiry, etc.) |
| **Affiliate**     | Link generation, click tracking, analytics                            |

## Prerequisites

- **Go** 1.24.0 or later
- **PostgreSQL** with [TimescaleDB](https://www.timescale.com/) extension
- **RabbitMQ** with management plugin (enable `rabbitmq_delayed_message_exchange` plugin)
- **Redis** or **Valkey**
- **AWS S3** bucket (or compatible storage)
- **FFmpeg** (included in Docker image, install manually for local dev)

## Configuration

### 1. Copy template files

```bash
cp config/config.yaml.template config/config.yaml
cp config/.env.template config/.env
```

### 2. Edit `config/config.yaml`

Configure database, RabbitMQ, Redis, S3, PayOS, GHN, and social API credentials. See comments in the template for each section.

### 3. Edit `config/.env`

Environment variables take precedence over `config.yaml` values. Required variables:

```bash
# Database
DATABASE_HOST, DATABASE_PORT, DATABASE_USER, DATABASE_PASSWORD, DATABASE_DBNAME

# Cache
CACHE_HOST, CACHE_PORT, CACHE_DB, CACHE_PASSWORD

# RabbitMQ
RABBITMQ_HOST, RABBITMQ_PORT, RABBITMQ_USERNAME, RABBITMQ_PASSWORD

# AWS S3
AWS_S3_BUCKET_ENDPOINT, AWS_S3_BUCKET_ACCESS_KEY, AWS_S3_BUCKET_SECRET_KEY

# Payments (PayOS)
PAYOS_CLIENT_ID, PAYOS_API_KEY, PAYOS_CHECKSUM_KEY

# Social APIs
SOCIAL_FACEBOOK_CLIENT_ID, SOCIAL_FACEBOOK_CLIENT_SECRET
SOCIAL_TIKTOK_CLIENT_KEY, SOCIAL_TIKTOK_CLIENT_SECRET

# AI Providers
AI_PROVIDERS_GEMINI_API_KEY, AI_PROVIDERS_OPENROUTER_API_KEY

# Notifications
GMAIL_SMTP_USERNAME, GMAIL_APP_PASSWORD
FIREBASE_SERVICE_ACCOUNT_PATH, FIREBASE_PROJECT_ID

# JWT (if using Vault)
JWT_VAULT_ENABLED, JWT_VAULT_ADDRESS, JWT_VAULT_TOKEN
```

## Installation & Running

### Install dependencies

```bash
go mod download
```

### Generate Swagger docs

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g ./cmd/server/main.go -o ./docs --parseInternal
```

### Run the application

```bash
go run ./cmd/server
```

The server starts on `http://localhost:8080` by default. Swagger UI is available at `http://localhost:8080/swagger/index.html`.

## Docker

```bash
# Build
docker build -t core-backend .

# Run
docker run -p 8080:8080 \
  -v $(pwd)/config:/app/config \
  core-backend
```

The Docker image is multi-stage (Alpine builder + scratch final) with FFmpeg bundled.

## Testing

Tests are organized under `tests/` with a Makefile for convenience.

```bash
# All tests
go test ./tests/... -v

# Unit tests only
go test ./tests/unit_tests/... -v

# Integration tests (requires running services)
go test ./tests/integration_tests/... -v

# Performance benchmarks
go test ./tests/performance_tests/... -bench=. -benchmem

# Coverage report
go test ./tests/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

Or use the Makefile in `tests/`:

```bash
cd tests && make test-report
```

## CLI Tools

### crypto_cli

Generate RSA key pairs for JWT signing:

```bash
go run ./cmd/crypto_cli
```

## Troubleshooting

- **TimescaleDB not installed**: See error "extension timescaledb does not exist"
  - Install TimescaleDB extension package for your PostgreSQL version
- **Hypertable conversion failed**: See error "table is already a hypertable"
  - This is safe — table was already converted in a previous migration
- **Compression not working**: Chunks remain uncompressed
  - Check compression policy exists and job is running:
    ```sql
    SELECT * FROM timescaledb_information.jobs WHERE proc_name LIKE '%compress%';
    ```
- **RabbitMQ delayed messages not working**: Enable the plugin:
  ```bash
  rabbitmq-plugins enable rabbitmq_delayed_message_exchange
  ```
