# Enterprise AI Assistant Foundation - Quick Start

## 🚀 Get Started in 5 Minutes

This guide helps you get the Enterprise AI Assistant Foundation running and start extending it immediately.

## 📋 Prerequisites

- **Go 1.21+** - [Download](https://golang.org/dl/)
- **Docker & Docker Compose** - [Install](https://docs.docker.com/get-docker/)
- **OpenAI API Key** - [Get one here](https://platform.openai.com/api-keys)

## ⚡ Quick Setup

### 1. Clone and Configure

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/Go_AI_Assistant.git
cd Go_AI_Assistant

# Copy environment configuration
cp .env.example .env

# Edit .env with your API keys
nano .env  # or use your favorite editor
```

### 2. Start Dependencies

```bash
# Start MongoDB and Redis
make up

# Initialize database
make migrate-up
```

### 3. Run the Application

```bash
# Start the AI assistant
make run
```

### 4. Verify Setup

```bash
# Test health endpoint
curl http://localhost:8080/health

# Should return:
# {
#   "status": "healthy",
#   "checks": {
#     "mongodb": "ok",
#     "redis": "ok"
#   }
# }
```

## 🎯 Your First Tool in 10 Minutes

Let's create a simple "calculator" tool that the AI can use.

### Step 1: Create Tool Directory

```bash
mkdir internal/tools/calculator
```

### Step 2: Create Tool Implementation

Create `internal/tools/calculator/calculator.go`:

```go
package calculator

import (
    "context"
    "fmt"
)

type CalculatorTool struct{}

func New() *CalculatorTool {
    return &CalculatorTool{}
}

func (c *CalculatorTool) Name() string {
    return "calculate"
}

func (c *CalculatorTool) Description() string {
    return "Perform basic arithmetic calculations"
}

func (c *CalculatorTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "operation": map[string]string{
                "type": "string",
                "enum": []string{"add", "subtract", "multiply", "divide"},
            },
            "a": map[string]string{"type": "number"},
            "b": map[string]string{"type": "number"},
        },
        "required": []string{"operation", "a", "b"},
    }
}

func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    operation := args["operation"].(string)
    a := args["a"].(float64)
    b := args["b"].(float64)
    
    var result float64
    switch operation {
    case "add":
        result = a + b
    case "subtract":
        result = a - b
    case "multiply":
        result = a * b
    case "divide":
        if b == 0 {
            return "", fmt.Errorf("cannot divide by zero")
        }
        result = a / b
    default:
        return "", fmt.Errorf("unknown operation: %s", operation)
    }
    
    return fmt.Sprintf("%.2f %s %.2f = %.2f", a, getSymbol(operation), b, result), nil
}

func getSymbol(op string) string {
    switch op {
    case "add": return "+"
    case "subtract": return "-"
    case "multiply": return "×"
    case "divide": return "÷"
    default: return "?"
    }
}
```

### Step 3: Register the Tool

Edit `internal/tools/factory/factory.go`:

```go
func (f *Factory) CreateAllTools() *registry.ToolRegistry {
    // ... existing tools
    f.registerCalculatorTool()  // Add this line
    return f.registry
}

func (f *Factory) registerCalculatorTool() {
    calculatorTool := calculator.New()
    f.registry.Register(calculatorTool)
}
```

### Step 4: Test Your Tool

```bash
# Restart the application
make run

# Test via API
curl -X POST http://localhost:8080/twirp/chat.ChatService/StartConversation \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Calculate 15 times 8"
  }'

# You should get a response using your new calculator tool!
```

## 🔍 Testing Your Changes

### Run Unit Tests

```bash
# Test your specific tool
go test ./internal/tools/calculator/...

# Run all tests
make test
```

### Integration Testing

```bash
# Test with real dependencies
make test-integration
```

### Manual Testing

```bash
# Start the application
make run

# Test different queries
curl -X POST http://localhost:8080/twirp/chat.ChatService/StartConversation \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is 100 divided by 25?"
  }'
```

## 🐛 Common Issues & Solutions

### Issue: "Connection refused" to MongoDB/Redis

**Solution:**

```bash
# Check if containers are running
docker ps

# If not, start them
make up

# Check logs
docker-compose logs mongodb
docker-compose logs redis
```

### Issue: OpenAI API errors

**Solution:**

- Verify your `OPENAI_API_KEY` in `.env`
- Check API key permissions and billing
- Test API key directly: `curl -H "Authorization: Bearer YOUR_KEY" https://api.openai.com/v1/models`

### Issue: Tool not discovered

**Solution:**

- Verify tool registration in `factory.go`
- Check tool name doesn't conflict with existing tools
- Restart the application after changes

### Issue: Database migration failures

**Solution:**

```bash
# Reset and retry
make migrate-down
make migrate-up

# Check migration status
make migrate-status
```

## 📚 Next Steps

### 1. Explore Existing Tools

Study the built-in tools to understand patterns:

- `internal/tools/weather/` - External API integration
- `internal/tools/datetime/` - Simple utility tool
- `internal/tools/holidays/` - Complex data processing

### 2. Learn Architecture

Read the [ARCHITECTURE.md](ARCHITECTURE.md) to understand the enterprise foundation design.

### 3. Extend Further

Follow the [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) for advanced extensions:

- Add middleware
- Create custom metrics
- Implement authentication
- Add caching and circuit breakers

### 4. Join the Community

- Check GitHub issues for common patterns
- Review pull requests for examples
- Contribute your own tools and improvements

## 🛠️ Development Workflow

### Daily Development

```bash
# Start fresh each day
make up
make migrate-up
make run

# Run tests before committing
make test

# Check code quality
make lint
```

### Debugging Tips

```bash
# Enable debug logging
export LOG_LEVEL=debug
make run

# Check application logs
tail -f logs/app.log

# Monitor metrics
curl http://localhost:8080/metrics
```

### Performance Testing

```bash
# Run benchmarks
make test-performance

# Load testing
make load-test
```

## 🎉 Congratulations

You've successfully:

- ✅ Set up the Enterprise AI Assistant Foundation
- ✅ Created and registered your first tool
- ✅ Tested the integration
- ✅ Learned the development workflow

Now you're ready to build enterprise-grade AI tools! Check out the full [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md) for advanced patterns and best practices.
