# Go AI Assistant

A sophisticated AI assistant built with Go, featuring dynamic prompt management, tool integration, and robust error handling.

## Features

- **Dynamic Prompt Management**: Prompts are stored in MongoDB with Redis caching for fast access
- **Tool Integration**: Weather, calendar, datetime, and other utility tools
- **Multi-platform Support**: Telegram, web, and API interfaces
- **Robust Error Handling**: Circuit breakers, retry logic, and graceful degradation
- **Performance Monitoring**: Comprehensive metrics and logging

## Architecture

### Prompt Management System

The system uses a multi-layered approach for prompt management:

1. **Redis Cache** - Fast access with TTL
2. **MongoDB Storage** - Persistent storage with versioning
3. **Fallback Prompts** - Hardcoded defaults for reliability

### Database Schema

```javascript
// prompt_configs collection
{
  _id: ObjectId,
  name: "title_generation" | "system_prompt" | "user_instruction",
  version: "v1",
  content: "Prompt text...",
  is_active: true,
  platform: "all" | "telegram" | "web",
  user_segment: "all" | "premium",
  created_at: ISODate(),
  updated_at: ISODate(),
  fallback_content: "Backup prompt"
}
```

## Setup

### Prerequisites

- Go 1.21+
- MongoDB 7.0+
- Redis 7.0+
- OpenAI API key

### Installation

1. Clone the repository:

```bash
git clone <repository-url>
cd tech-challenge
```

2. Install dependencies:

```bash
go mod download
```

3. Set up environment variables:

```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Start services with Docker:

```bash
docker-compose up -d
```

5. **IMPORTANT: Run database migrations before first start**:

```bash
# Apply initial migrations
docker exec -it mongodb mongosh --eval "load('/data/configdb/migrations/000001_init.up.js')"
docker exec -it mongodb mongosh --eval "load('/data/configdb/migrations/000002_prompts_init.js')"
```

### Environment Variables

```bash
OPENAI_API_KEY=your_openai_api_key
OPENAI_MODEL=gpt-4o-mini
WEATHER_API_KEY=your_weather_api_key
REDIS_ADDR=localhost:6379
MONGO_URI=mongodb://acai:travel@localhost:27017
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
API_KEY=your_api_key_for_protection
```

## Usage

### Running the Application

```bash
# Start the server
go run cmd/server/main.go

# Or use CLI
go run cmd/cli/main.go
```

### Managing Prompts

Prompts are dynamically managed through MongoDB. You can update prompts without restarting the application:

```javascript
// Example: Update system prompt
db.prompt_configs.updateOne(
  { name: "system_prompt", version: "v1" },
  { 
    $set: { 
      content: "New improved system prompt...",
      updated_at: new Date()
    }
  }
);
```

### API Endpoints

- `POST /chat/conversation` - Create new conversation
- `GET /chat/conversation/{id}` - Get conversation
- `POST /chat/conversation/{id}/message` - Send message
- `GET /health` - Health check

## Development

### Project Structure

```
internal/
├── chat/           # Core chat functionality
├── config/         # Configuration management
├── mongox/         # MongoDB utilities
├── redisx/         # Redis caching
├── tools/          # AI assistant tools
└── metrics/        # Performance monitoring
```

### Adding New Tools

1. Create tool in `internal/tools/`
2. Register in `internal/tools/factory/factory.go`
3. Update tool registry

### Testing

```bash
# Run all tests
go test ./...

# Run specific test suite
go test ./tests/unit/...
go test ./tests/integration/...
```

## Security Features

- **Prompt Injection Protection**: Enhanced system prompts with security instructions
- **API Rate Limiting**: Configurable rate limits for API endpoints
- **Circuit Breakers**: Automatic service degradation on failures
- **Input Validation**: Comprehensive input sanitization

## Monitoring

The application provides:

- Structured logging with context
- Performance metrics (token usage, response times)
- Health checks for all dependencies
- Error tracking and reporting

## Troubleshooting

### Common Issues

1. **Prompts not loading**: Ensure MongoDB migration `000002_prompts_init.js` has been executed
2. **Redis connection issues**: Check Redis is running and accessible
3. **OpenAI API errors**: Verify API key and model configuration

### Health Checks

```bash
# Check system health
curl http://localhost:8080/health
```

## License

MIT License
