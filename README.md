# Go AI Assistant

A production-ready AI assistant backend built with Go, featuring modular tool architecture, Redis caching, OpenTelemetry observability, and comprehensive monitoring. Supports OpenAI/Anthropic APIs with intelligent caching, rate limiting, and circuit breaker patterns for external services.

![CI](https://github.com/YOUR_USERNAME/Go_AI_Assistant/workflows/CI/badge.svg)

## Key Features

- 🤖 **AI Integration** - OpenAI/Anthropic API support with configurable models
- 🔧 **Modular Tools** - Weather API, calendar, date/time tools with plugin architecture
- 💾 **Redis Caching** - SHA256-hashed keys for security, 24h TTL, cache hit/miss tracking
- 📊 **Observability** - OpenTelemetry tracing, Prometheus metrics, structured logging
- 🔒 **Security** - API key authentication, per-IP rate limiting, constant-time comparison
- 🛡️ **Resilience** - Circuit breaker for external APIs, graceful degradation
- 🗄️ **MongoDB Storage** - Conversation history with optimized indexes
- ✅ **Testing** - Unit, integration, E2E, and performance tests (75%+ coverage)
- 🚀 **Production Ready** - Health checks, migrations, backups, CI/CD pipeline

## Tech Stack

- **Backend**: Go 1.21+, Twirp RPC
- **Database**: MongoDB 7
- **Cache**: Redis 7
- **Observability**: OpenTelemetry, Prometheus
- **Infrastructure**: Docker, Docker Compose
- **CI/CD**: GitHub Actions

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21 or higher
- (Optional) OpenAI API key

### Setup

1. **Clone and configure**

   ```bash
   git clone https://github.com/YOUR_USERNAME/Go_AI_Assistant.git
   cd Go_AI_Assistant
   cp .env.example .env
   # Edit .env with your API keys
   ```

2. **Start services**

   ```bash
   docker-compose up -d
   make migrate-up  # Create database indexes
   ```

3. **Run application**

   ```bash
   go run ./cmd/server
   ```

4. **Verify deployment**

   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/metrics
   ```

### API Endpoints

- `GET /health` - Health check (MongoDB + Redis status)
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics (requires API key)
- `POST /twirp/chat.ChatService/*` - Chat API (Twirp RPC)

## Testing

```bash
# Run all tests
make test

# Specific test suites
make test-unit          # Unit tests
make test-integration   # Integration tests (requires Docker)
make test-e2e          # End-to-end tests
make test-performance  # Benchmarks

# Coverage report
make test-coverage

# Smoke test (full system validation)
make smoke
```

## Configuration

Key environment variables (see `.env.example`):

```bash
# Required
OPENAI_API_KEY=sk-...              # OpenAI API key
MONGO_URI=mongodb://...             # MongoDB connection string
REDIS_ADDR=localhost:6379           # Redis address

# Optional
OPENAI_MODEL=gpt-4o-mini           # AI model selection
WEATHER_API_KEY=...                 # Weather API key
API_KEY=...                         # API authentication key
API_RPS=10.0                        # Rate limit (requests/second)
API_BURST=20                        # Rate limit burst size
CIRCUIT_BREAKER_MAX_FAILURES=3      # Circuit breaker threshold
CIRCUIT_BREAKER_COOLDOWN_SECONDS=30 # Circuit breaker cooldown
```

## Monitoring

**Prometheus Metrics** (`/metrics`):

- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request latency
- `http_requests_in_flight` - Active requests
- `openai_tokens_input_total` - OpenAI token usage
- `weather_circuit_state` - Circuit breaker status

**Health Checks**:

```bash
curl http://localhost:8080/health | jq
# {
#   "status": "healthy",
#   "checks": {
#     "mongodb": "ok",
#     "redis": "ok"
#   }
# }
```

**Logs**: Structured JSON with trace IDs for request correlation

## Database Management

```bash
# Migrations
make migrate-up      # Apply migrations
make migrate-down    # Rollback migrations
make migrate-status  # Check current state

# Backups
make backup                              # Create timestamped backup
make restore BACKUP_PATH=backups/...    # Restore from backup
```

## Project Structure

```
├── cmd/
│   ├── server/          # Main application
│   └── cli/             # CLI tools
├── internal/
│   ├── chat/            # Chat service implementation
│   ├── circuitbreaker/  # Circuit breaker pattern
│   ├── config/          # Configuration management
│   ├── errorsx/         # Error handling utilities
│   ├── health/          # Health check endpoints
│   ├── httpx/           # HTTP middleware (auth, rate limit)
│   ├── metrics/         # Prometheus metrics
│   ├── redisx/          # Redis cache layer
│   └── tools/           # Modular tool system
├── migrations/          # Database migrations
├── scripts/             # Utility scripts
└── tests/              # Test suites
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

**Code Quality**: All PRs must pass CI checks (tests, linting, formatting)

## License

MIT License (see LICENSE file)

## Links

- [Production Readiness Checklist](PRODUCTION_READINESS.md)
- [Architecture Docs](docs/)
- [API Documentation](https://github.com/YOUR_USERNAME/Go_AI_Assistant/wiki)
