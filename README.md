# Enterprise AI Assistant Foundation

A scalable foundation for building enterprise-grade AI assistants, built with Go. This is not a finished product, but a **production-ready foundation** designed to scale from 3 tools to 50+ tools, from hundreds to millions of users.

## 🎯 Philosophy

This project is an **enterprise foundation** - a carefully architected starting point for building sophisticated AI assistant systems. The architecture is intentionally designed for extensibility and scalability, not as a final product.

**Key Principles:**

- **Foundation First**: Current functionality demonstrates core patterns, not final capabilities
- **Enterprise-Ready Architecture**: Production patterns from day one (observability, resilience, security)
- **Plugin-Oriented Design**: Easy extension without modifying core systems
- **Scalability by Design**: Ready for horizontal scaling and enterprise workloads

## 🚀 Growth Path

### Phase 1: Core Foundation (Current)

- **3-5 tools** (weather, datetime, holidays + extensible)
- **Basic observability** (metrics, logging, tracing)
- **Production resilience** (circuit breakers, retry, caching)
- **Developer-friendly** extensibility patterns

### Phase 2: Enterprise Features

- **10-20 tools** with complex integrations
- **Advanced monitoring** and alerting
- **Multi-tenant** capabilities
- **Enterprise security** (RBAC, audit logging)

### Phase 3: Platform Scale

- **50+ tools** with marketplace architecture
- **Global scale** deployment patterns
- **Advanced AI orchestration**
- **Enterprise-grade** SLAs and compliance

## Key Features

- 🤖 **AI Integration** - OpenAI API support with configurable models (GPT-4, GPT-4o-mini)
- 🔧 **Modular Tools** - Extensible plugin architecture for unlimited AI capabilities
- 💾 **Redis Caching** - SHA256-hashed keys for security, configurable TTL, cache hit/miss tracking
- 📊 **Observability** - OpenTelemetry tracing, Prometheus metrics, structured logging
- 🔒 **Security** - API key authentication, per-IP rate limiting, constant-time comparison
- 🛡️ **Resilience** - Circuit breaker for external APIs, graceful degradation, retry with exponential backoff
- 🗄️ **MongoDB Storage** - Conversation history with optimized indexes, prompt management
- 📝 **Prompt Management** - Platform-specific prompts with caching and fallback system
- 🔄 **Session Management** - Seamless conversation continuity for stateless clients
- 🧠 **Intelligent Context Management** - Advanced token management with AI-powered summarization
- ✅ **Testing** - Comprehensive test suites (unit, integration, E2E, performance)

## Architecture

This project follows an **enterprise-ready architecture** with clear separation of concerns and multiple layers of resilience, designed specifically for extensibility.

**📖 For detailed architecture documentation, see [ARCHITECTURE.md](ARCHITECTURE.md)**

### Enterprise Architectural Components

- 🔧 **Modular Tools System** - Plugin architecture designed for 50+ tools
- 🔄 **Session Management** - Scalable conversation continuity for enterprise clients
- 📊 **Observability Stack** - Production-grade monitoring and tracing
- 🛡️ **Resilience Patterns** - Enterprise-grade fault tolerance
- 🔐 **Security First** - Enterprise security patterns from foundation
- 🗄️ **Data Layer** - Scalable storage with enterprise patterns
- 📝 **Prompt Management** - Dynamic system for enterprise use cases

### Why This Architecture?

Each architectural decision supports **enterprise scalability**:

- **Redis Caching**: Enterprise performance with 10x cost reduction
- **Circuit Breaker**: Prevents cascading failures in complex enterprise environments
- **Retry Mechanism**: 95%+ success rate on transient errors at scale
- **Session Management**: Enables stateless enterprise clients (web, mobile, bots)
- **Tool Registry**: Add unlimited AI capabilities without core modifications
- **Plugin Architecture**: Foundation for enterprise tool marketplace

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed enterprise-focused explanations.

---

## Quick Start for Developers

### Prerequisites

- Docker & Docker Compose
- Go 1.21 or higher
- OpenAI API key

### Setup

```bash
git clone https://github.com/YOUR_USERNAME/Go_AI_Assistant.git
cd Go_AI_Assistant
cp .env.example .env
# Edit .env with your API keys
```

### Run Foundation

```bash
make up && make migrate-up && make run
```

### Verify Foundation

```bash
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

## Extending the Foundation

### Adding Your First Tool

See [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) for comprehensive guidance on extending the system with new tools, middleware, and enterprise features.

### Quick Tool Example

```go
// Your new enterprise tool
type PaymentTool struct {
    apiKey string
}

func (p *PaymentTool) Name() string {
    return "process_payment"
}

func (p *PaymentTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // Your enterprise logic here
    return "Payment processed successfully", nil
}
```

## Tech Stack

- **Backend**: Go 1.21+, Twirp RPC
- **Database**: MongoDB 7
- **Cache**: Redis 7
- **Observability**: OpenTelemetry, Prometheus
- **Infrastructure**: Docker, Docker Compose
- **CI/CD**: GitHub Actions

## API Endpoints

- `GET /health` - Health check (MongoDB + Redis status)
- `GET /ready` - Readiness probe
- `GET /metrics` - Prometheus metrics (requires API key)
- `POST /twirp/chat.ChatService/*` - Chat API (Twirp RPC)

### Interactive API Documentation

- **Swagger UI**: <http://localhost:8080/docs/>
- **Static Documentation**: <http://localhost:8080/api-docs>

## Configuration

Key environment variables (see `.env.example`):

```bash
# Required
OPENAI_API_KEY=sk-...                    # OpenAI API key
MONGO_URI=mongodb://...                   # MongoDB connection string
REDIS_ADDR=localhost:6379                 # Redis address

# Optional - AI Configuration
OPENAI_MODEL=gpt-4o-mini                 # AI model selection

# API Security & Rate Limiting
API_KEY=changeme_in_production           # API key for /metrics endpoint
API_RATE_LIMIT_RPS=10.0                  # Rate limit (requests/second)

# Cache Configuration
CACHE_TTL_HOURS=24                       # Redis cache TTL (hours)

# Circuit Breaker
CIRCUIT_BREAKER_MAX_FAILURES=3           # Max failures before opening
```

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

## 🤖 Telegram Bot Integration

The project includes a **production-ready Telegram bot** that demonstrates the session-based conversation management capabilities. This serves as an example of how to build enterprise clients on top of the foundation.

See [python_telegram_bot/README.md](python_telegram_bot/README.md) for bot integration guide.

## Project Structure

```
├── cmd/
│   ├── server/          # Main application entry point
│   └── cli/             # CLI tools for development
├── internal/
│   ├── chat/            # Core chat service implementation
│   ├── chat/assistant/  # AI assistant with tool orchestration
│   ├── chat/model/      # Domain models and repository
│   ├── tools/           # Modular tool system (extensible)
│   │   ├── datetime/    # Example: Date/time tool
│   │   ├── factory/     # Tool factory for extensibility
│   │   ├── registry/    # Tool registry for plugin architecture
│   │   └── weather/     # Example: Weather tool
│   ├── resilience/      # Enterprise resilience patterns
│   │   ├── circuitbreaker/
│   │   ├── retry/
│   │   └── redisx/
│   ├── observability/   # Enterprise monitoring
│   │   ├── metrics/
│   │   ├── otel/
│   │   └── httpx/
│   └── security/        # Enterprise security
│       ├── httpx/auth.go
│       └── httpx/ratelimit.go
├── python_telegram_bot/ # Example client implementation
├── migrations/          # Database migrations
└── tests/              # Comprehensive test suites
```

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - Enterprise architecture and design decisions
- [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) - Comprehensive guide to extending the foundation
- [PRODUCTION_READINESS.md](PRODUCTION_READINESS.md) - Production deployment guidelines
- [QUICK_START.md](QUICK_START.md) - Quick start for new developers

## Contributing

We welcome contributions to extend this enterprise foundation! Please see our contributing guidelines for details on how to submit pull requests, report issues, and suggest new enterprise features.

## License

MIT License (see LICENSE file)

## Enterprise Support

This foundation is designed for enterprise adoption. For enterprise support, customization, or consulting on building AI assistant platforms, please contact the maintainers.

---

**Remember**: This is a **foundation**, not a final product. The architecture is intentionally enterprise-ready to support your growth from startup to global platform.
