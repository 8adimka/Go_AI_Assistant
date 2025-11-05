# Go AI Assistant

A production-ready AI assistant service built with Go, featuring OpenAI GPT-4 integration, real-time weather data, and Telegram bot interface. This project demonstrates enterprise-grade software development practices with comprehensive testing, monitoring, and scalability features.

## 🚀 Features

### Core AI Assistant

- 🤖 **OpenAI GPT-4 Integration** - Intelligent conversation handling with smart prompts
- 💬 **Conversation Management** - Persistent conversations with MongoDB storage
- 📝 **Smart Title Generation** - Automatic conversation summarization with Redis caching

### Real-time Data Services

- 🌤️ **Weather Information** - Real-time weather data via WeatherAPI.com
- 📅 **Holiday Information** - Local bank and public holidays
- ⏰ **Current Date/Time** - Real-time temporal information

### Production Infrastructure

- 💾 **Redis Caching** - Performance optimization for API calls and prompts
- 🔄 **Rate Limiting** - Protection against API abuse using `golang.org/x/time/rate`
- 🛡️ **Graceful Shutdown** - Proper cleanup and connection handling
- 🔄 **Retry Logic** - Exponential backoff for external API calls

### Monitoring & Observability

- 📊 **OpenTelemetry** - Distributed tracing and metrics collection
- 📈 **Prometheus Metrics** - Request count, latency, error rates
- 🔍 **Jaeger Tracing** - Request flow visualization
- 📝 **Structured Logging** - Contextual logging with trace IDs

### User Interfaces

- 🌐 **RESTful API** - Twirp-based HTTP API with JSON
- 🤖 **Telegram Bot** - User-friendly chat interface
- 🖥️ **CLI Tool** - Command-line interface for testing

## 🏗️ Architecture

```
tech-challenge/              # Go AI Assistant (Go project)
├── cmd/                    # Go entry points (server, CLI)
├── internal/               # Private application code
│   ├── chat/              # Conversation management
│   ├── assistant/         # AI assistant logic
│   ├── weather/           # Weather service integration
│   ├── config/            # Configuration management
│   └── httpx/             # HTTP utilities
├── python_telegram_bot/    # Telegram bot interface
│   ├── telegram_bot_enhanced.py
│   ├── requirements.txt
│   ├── .env
│   └── README.md
├── rpc/                    # Protocol buffers definitions
├── docker-compose.yaml     # Infrastructure (Redis, MongoDB)
├── go.mod                  # Go dependencies
├── go.sum                  # Dependency checksums
├── Makefile                # Build and development commands
└── README.md              # This file
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- OpenAI API Key
- WeatherAPI.com Key (optional, fallback available)

### 1. Clone and Setup

```bash
git clone https://github.com/8adimka/Go_AI_Assistant.git
cd Go_AI_Assistant
```

### 2. Environment Configuration

```bash
# Copy environment template
cp .env.example .env

# Edit .env file with your API keys
nano .env
```

Required environment variables:

```env
OPENAI_API_KEY=your_openai_api_key_here
WEATHER_API_KEY=your_weatherapi_key_here  # Optional, fallback available
```

### 3. Start Infrastructure

```bash
# Start Redis and MongoDB
docker-compose up -d

# Verify services are running
docker-compose ps
```

### 4. Run the Application

```bash
# Start the Go server
go run ./cmd/server
```

You should see:

```
2025/11/05 15:10:46 INFO Successfully connected to Redis addr=localhost:6379
2025/11/05 15:10:46 INFO Starting the server... port=8080
```

## 📡 API Usage

### Start a Conversation

```bash
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/StartConversation \
  -H "Content-Type: application/json" \
  -d '{"message": "What is the weather like in Barcelona?"}' | jq
```

Response:

```json
{
  "conversation_id": "690b5b2b1ce3d57c09602ecf",
  "title": "Weather in Barcelona",
  "reply": "The current weather in Barcelona is..."
}
```

### Send Message to Conversation

```bash
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/SendMessage \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "690b5b2b1ce3d57c09602ecf",
    "message": "What about tomorrow?"
  }' | jq
```

### Get Conversation

```bash
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/GetConversation \
  -H "Content-Type: application/json" \
  -d '{"conversation_id": "690b5b2b1ce3d57c09602ecf"}' | jq
```

### List Conversations

```bash
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/ListConversations \
  -H "Content-Type: application/json" \
  -d '{}' | jq
```

## 🤖 Telegram Bot

### Setup

```bash
# Navigate to bot directory
cd python_telegram_bot

# Create virtual environment
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt

# Configure environment
cp .env.example .env
# Edit .env with your Telegram Bot Token
```

### Run the Bot

```bash
source venv/bin/activate
python telegram_bot_enhanced.py
```

### Bot Commands

- `/start` - Welcome message and bot description
- `/status` - Check system status and connectivity
- `/weather <city>` - Get weather for specified city
- **Any message** - Send to AI assistant for response

## 🧪 Testing

### Run All Tests

```bash
# Make sure infrastructure is running
docker-compose up -d

# Run tests
go test ./...
```

### Test Categories

- **Unit Tests**: `go test ./internal/assistant/...`
- **Integration Tests**: `go test ./internal/chat/...`
- **E2E Tests**: Tests with real external services

### Test Coverage

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📊 Monitoring & Observability

### Metrics Endpoint

```bash
# Prometheus metrics
curl http://localhost:8080/metrics
```

### Health Checks

```bash
# Health endpoint
curl http://localhost:8080/healthz

# Readiness endpoint
curl http://localhost:8080/readyz
```

### Jaeger Tracing

Start Jaeger for distributed tracing:

```bash
docker run -d -p 16686:16686 -p 4317:4317 jaegertracing/all-in-one:latest
```

Access Jaeger UI at: <http://localhost:16686>

## 🔧 Development

### Building

```bash
# Build the application
go build ./cmd/server

# Build for production
GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run linters
golangci-lint run

# Run vet
go vet ./...
```

### Make Commands

```bash
make up      # Start infrastructure (Redis, MongoDB)
make down    # Stop infrastructure
make run     # Start application
make test    # Run tests
make gen     # Generate protobuf code
```

## 🏭 Production Deployment

### Docker

```bash
# Build Docker image
docker build -t go-ai-assistant .

# Run with Docker
docker run -p 8080:8080 --env-file .env go-ai-assistant
```

### Environment Variables for Production

```env
OPENAI_API_KEY=your_production_key
WEATHER_API_KEY=your_weatherapi_key
REDIS_URL=redis://redis:6379
MONGODB_URL=mongodb://mongo:27017
LOG_LEVEL=info
```

## 🔒 Security Features

- **Input Validation** - All inputs are validated
- **Rate Limiting** - Prevents API abuse
- **CORS Headers** - Configurable CORS policies
- **Secrets Management** - Environment-based configuration
- **HTTPS Ready** - Prepared for TLS termination

## 📈 Performance

- **Redis Caching** - Reduces OpenAI API calls
- **Connection Pooling** - Optimized HTTP clients
- **Async Processing** - Non-blocking operations
- **Memory Optimization** - Efficient data structures

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

### Development Guidelines

- Write tests for new features
- Follow Go conventions and style guide
- Update documentation for API changes
- Use meaningful commit messages

## 📄 License

This project is part of a technical challenge for Acai Travel.

## 🆘 Troubleshooting

### Common Issues

**Bot not starting:**

- Check TELEGRAM_BOT_TOKEN in .env file
- Verify virtual environment is activated

**API connection errors:**

- Ensure Redis and MongoDB are running: `docker-compose ps`
- Check OpenAI API key is valid

**Weather API failures:**

- System has fallback to mock data
- Verify WeatherAPI key if real data is needed

**Port conflicts:**

- Change port in configuration if 8080 is occupied

### Getting Help

- Check application logs for detailed error messages
- Verify all environment variables are set
- Ensure all dependencies are installed

---

**Ready to build intelligent conversations?** 🚀
