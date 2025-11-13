# Enterprise AI Assistant Foundation - Developer Guide

## 🎯 Overview

This guide provides comprehensive instructions for extending the Enterprise AI Assistant Foundation. The architecture is designed for unlimited extensibility while maintaining enterprise-grade reliability and performance.

## 🚀 Quick Start

### Prerequisites

- Go 1.21+ installed
- Docker and Docker Compose
- Basic understanding of Go interfaces and dependency injection

### Setup Development Environment

```bash
# Clone and setup
git clone https://github.com/YOUR_USERNAME/Go_AI_Assistant.git
cd Go_AI_Assistant
cp .env.example .env

# Start dependencies
make up

# Run tests to verify setup
make test
```

## 🔧 Extending the System

### Adding a New Tool

Tools are the primary extension point. Each tool implements the `Tool` interface and is automatically discovered by the system.

#### Step 1: Create Tool Implementation

Create a new directory in `internal/tools/` for your tool:

```bash
mkdir internal/tools/calculator
```

Create `internal/tools/calculator/calculator.go`:

```go
package calculator

import (
    "context"
    "fmt"
    "strconv"
)

// CalculatorTool performs basic arithmetic operations
type CalculatorTool struct{}

// New creates a new CalculatorTool instance
func New() *CalculatorTool {
    return &CalculatorTool{}
}

// Name returns the tool name (used by OpenAI)
func (c *CalculatorTool) Name() string {
    return "calculate"
}

// Description provides the tool description for OpenAI
func (c *CalculatorTool) Description() string {
    return "Perform basic arithmetic calculations (addition, subtraction, multiplication, division)"
}

// Parameters defines the JSON schema for tool parameters
func (c *CalculatorTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "operation": map[string]string{
                "type": "string",
                "enum": []string{"add", "subtract", "multiply", "divide"},
                "description": "The arithmetic operation to perform",
            },
            "a": map[string]string{
                "type": "number",
                "description": "First number",
            },
            "b": map[string]string{
                "type": "number",
                "description": "Second number",
            },
        },
        "required": []string{"operation", "a", "b"},
    }
}

// Execute performs the calculation
func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    operation := args["operation"].(string)
    a := args["a"].(float64)
    b := args["b"].(float64)
    
    var result float64
    var err error
    
    switch operation {
    case "add":
        result = a + b
    case "subtract":
        result = a - b
    case "multiply":
        result = a * b
    case "divide":
        if b == 0 {
            return "", fmt.Errorf("division by zero")
        }
        result = a / b
    default:
        return "", fmt.Errorf("unknown operation: %s", operation)
    }
    
    return fmt.Sprintf("%.2f %s %.2f = %.2f", a, getOperationSymbol(operation), b, result), nil
}

// getOperationSymbol returns the symbol for the operation
func getOperationSymbol(operation string) string {
    switch operation {
    case "add":
        return "+"
    case "subtract":
        return "-"
    case "multiply":
        return "×"
    case "divide":
        return "÷"
    default:
        return "?"
    }
}
```

#### Step 2: Register Tool in Factory

Update `internal/tools/factory/factory.go`:

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

#### Step 3: Test Your Tool

Create `internal/tools/calculator/calculator_test.go`:

```go
package calculator

import (
    "context"
    "testing"
)

func TestCalculatorTool(t *testing.T) {
    tool := New()
    
    tests := []struct {
        name      string
        operation string
        a         float64
        b         float64
        want      string
        wantErr   bool
    }{
        {
            name:      "addition",
            operation: "add",
            a:         5,
            b:         3,
            want:      "5.00 + 3.00 = 8.00",
            wantErr:   false,
        },
        {
            name:      "division by zero",
            operation: "divide",
            a:         10,
            b:         0,
            want:      "",
            wantErr:   true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            args := map[string]interface{}{
                "operation": tt.operation,
                "a":         tt.a,
                "b":         tt.b,
            }
            
            got, err := tool.Execute(context.Background(), args)
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Execute() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

Run the test:

```bash
go test ./internal/tools/calculator/...
```

#### Step 4: Verify Tool Discovery

The tool is now automatically available to the AI assistant. Test it:

```bash
# Start the application
make run

# Test via API
curl -X POST http://localhost:8080/twirp/chat.ChatService/StartConversation \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Calculate 15 plus 27"
  }'
```

### Enterprise Tool Patterns

#### Pattern 1: External API Integration

For tools that call external APIs, follow this pattern:

```go
type APITool struct {
    client     *http.Client
    apiKey     string
    baseURL    string
    cache      *redisx.Cache
    circuit    *circuitbreaker.CircuitBreaker
}

func NewAPITool(apiKey string, cache *redisx.Cache) *APITool {
    return &APITool{
        client:  &http.Client{Timeout: 30 * time.Second},
        apiKey:  apiKey,
        baseURL: "https://api.example.com",
        cache:   cache,
        circuit: circuitbreaker.New(3, 30*time.Second),
    }
}

func (a *APITool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // Generate cache key
    cacheKey := a.cache.GenerateKey("api", fmt.Sprintf("%v", args))
    
    // Check cache first
    var cachedResult string
    if err := a.cache.Get(ctx, cacheKey, &cachedResult); err == nil {
        return cachedResult, nil
    }
    
    // Use circuit breaker for API call
    result, err := a.circuit.Execute(func() (interface{}, error) {
        return a.callExternalAPI(ctx, args)
    })
    
    if err != nil {
        return "", err
    }
    
    // Cache successful result
    if err := a.cache.Set(ctx, cacheKey, result); err != nil {
        // Log cache error but don't fail the request
        slog.WarnContext(ctx, "Failed to cache API result", "error", err)
    }
    
    return result.(string), nil
}
```

#### Pattern 2: Database-Backed Tool

For tools that need persistent storage:

```go
type DatabaseTool struct {
    repo *model.Repository
}

func (d *DatabaseTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // Use repository pattern for database operations
    data, err := d.repo.FindByCriteria(ctx, args)
    if err != nil {
        return "", fmt.Errorf("database error: %w", err)
    }
    
    return formatData(data), nil
}
```

#### Pattern 3: Complex Tool with Multiple Operations

For tools with multiple related functions:

```go
type MultiFunctionTool struct {
    operations map[string]func(ctx context.Context, args map[string]interface{}) (string, error)
}

func (m *MultiFunctionTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    operation := args["operation"].(string)
    
    if fn, exists := m.operations[operation]; exists {
        return fn(ctx, args)
    }
    
    return "", fmt.Errorf("unknown operation: %s", operation)
}
```

## 🏗️ Architecture Extension

### Using Consolidated Security Middleware

The system now provides a consolidated security middleware that combines authentication and rate limiting for simpler configuration.

#### Using Consolidated Security Middleware

```go
import "github.com/8adimka/Go_AI_Assistant/internal/httpx"

// Configure security middleware
securityConfig := httpx.SecurityConfig{
    APIKey:         cfg.APIKey,
    RateLimitRPS:   cfg.APIRateLimitRPS,
    RateLimitBurst: cfg.APIRateLimitBurst,
    PublicPaths:    []string{"/health", "/ready", "/docs", "/api-docs"},
}

securityMiddleware := httpx.NewSecurityMiddleware(securityConfig)

// Use in middleware chain
handler = securityMiddleware.Middleware()(handler)
```

#### Legacy Individual Middleware (Still Supported)

For fine-grained control, you can still use individual middleware:

```go
// Individual middleware (legacy approach)
handler = httpx.ProtectedRoutes(cfg.APIKey, publicPaths)(handler)
handler = httpx.RateLimit(cfg.APIRateLimitRPS, cfg.APIRateLimitBurst)(handler)
```

### Adding New Middleware

Middleware components process HTTP requests before they reach the main handlers.

#### Step 1: Create Middleware

Create `internal/httpx/audit.go`:

```go
package httpx

import (
    "log/slog"
    "net/http"
    "time"
)

// AuditMiddleware logs detailed request information for compliance
type AuditMiddleware struct {
    enabled bool
}

// NewAuditMiddleware creates a new audit middleware
func NewAuditMiddleware(enabled bool) *AuditMiddleware {
    return &AuditMiddleware{enabled: enabled}
}

// Middleware returns the HTTP middleware function
func (a *AuditMiddleware) Middleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !a.enabled {
                next.ServeHTTP(w, r)
                return
            }
            
            start := time.Now()
            
            // Log request details
            slog.InfoContext(r.Context(), "Audit request",
                "method", r.Method,
                "path", r.URL.Path,
                "user_agent", r.UserAgent(),
                "ip", GetClientIP(r),
                "timestamp", start,
            )
            
            next.ServeHTTP(w, r)
            
            // Log response details
            duration := time.Since(start)
            slog.InfoContext(r.Context(), "Audit response",
                "method", r.Method,
                "path", r.URL.Path,
                "duration_ms", duration.Milliseconds(),
            )
        })
    }
}
```

#### Step 2: Register Middleware

Update `cmd/server/main.go`:

```go
func main() {
    // ... existing setup
    
    // Add audit middleware
    auditMiddleware := httpx.NewAuditMiddleware(true)
    
    // Update middleware chain
    handler := setupMiddleware(router, cfg, auditMiddleware)
    
    // ... rest of main
}

func setupMiddleware(router *mux.Router, cfg *config.Config, auditMiddleware *httpx.AuditMiddleware) http.Handler {
    handler := httpx.HTTPMetrics(router)
    handler = otelHandler(handler)
    handler = httpx.Logger(handler)
    handler = auditMiddleware.Middleware()(handler)  // Add audit middleware
    handler = httpx.Recovery(handler)
    
    // Use consolidated security middleware
    securityConfig := httpx.SecurityConfig{
        APIKey:         cfg.APIKey,
        RateLimitRPS:   cfg.APIRateLimitRPS,
        RateLimitBurst: cfg.APIRateLimitBurst,
        PublicPaths:    []string{"/health", "/ready", "/docs", "/api-docs"},
    }
    securityMiddleware := httpx.NewSecurityMiddleware(securityConfig)
    handler = securityMiddleware.Middleware()(handler)
    
    return handler
}
```

### Extending Metrics

Add custom business metrics:

#### Step 1: Define Custom Metrics

Update `internal/metrics/metrics.go`:

```go
type Metrics struct {
    // ... existing metrics
    
    // Custom business metrics
    ToolUsageTotal        *prometheus.CounterVec
    UserActivityTotal     *prometheus.CounterVec
    BusinessKPI           *prometheus.GaugeVec
}

func New() *Metrics {
    metrics := &Metrics{
        // ... existing metric initialization
        
        ToolUsageTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "tool_usage_total",
                Help: "Total usage of each tool",
            },
            []string{"tool_name", "user_id", "platform"},
        ),
        
        UserActivityTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "user_activity_total",
                Help: "Total user activity events",
            },
            []string{"user_id", "activity_type"},
        ),
    }
    
    // Register new metrics
    prometheus.MustRegister(metrics.ToolUsageTotal)
    prometheus.MustRegister(metrics.UserActivityTotal)
    
    return metrics
}

// RecordToolUsage records when a tool is used
func (m *Metrics) RecordToolUsage(ctx context.Context, toolName, userID, platform string) {
    if m.ToolUsageTotal != nil {
        m.ToolUsageTotal.WithLabelValues(toolName, userID, platform).Inc()
    }
}
```

#### Step 2: Use Custom Metrics

In your tool implementation:

```go
func (t *YourTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // Record tool usage
    if metrics := metrics.FromContext(ctx); metrics != nil {
        metrics.RecordToolUsage(ctx, t.Name(), getUserID(ctx), getPlatform(ctx))
    }
    
    // ... tool logic
}
```

## 🔒 Security Extensions

### Adding Authentication

Implement custom authentication:

```go
type JWTAuth struct {
    secretKey []byte
}

func NewJWTAuth(secretKey string) *JWTAuth {
    return &JWTAuth{secretKey: []byte(secretKey)}
}

func (j *JWTAuth) Middleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            if token == "" {
                http.Error(w, "Authorization header required", http.StatusUnauthorized)
                return
            }
            
            // Validate JWT token
            userID, err := j.validateToken(token)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            // Add user to context
            ctx := context.WithValue(r.Context(), "userID", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

## 🧪 Testing Extensions

### Integration Testing

Create comprehensive integration tests:

```go
func TestToolIntegration(t *testing.T) {
    // Setup test environment
    cfg := config.Load()
    redisClient := redisx.MustConnect(cfg.RedisAddr)
    defer redisClient.Close()
    
    // Create tool with dependencies
    cache := redisx.NewCache(redisClient, time.Hour)
    tool := NewYourTool(cache)
    
    // Test tool execution
    result, err := tool.Execute(context.Background(), map[string]interface{}{
        "param1": "value1",
        "param2": "value2",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
}
```

### Performance Testing

Add performance benchmarks:

```go
func BenchmarkToolExecution(b *testing.B) {
    tool := NewYourTool(nil)
    args := map[string]interface{}{
        "param1": "test",
        "param2": 123,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = tool.Execute(context.Background(), args)
    }
}
```

## 📊 Monitoring and Observability

### Custom Dashboards

Create Grafana dashboards for your tools:

```json
{
  "dashboard": {
    "title": "Tool Performance",
    "panels": [
      {
        "title": "Tool Usage",
        "targets": [
          {
            "expr": "rate(tool_usage_total[5m])",
            "legendFormat": "{{tool_name}}"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

Define Prometheus alerting rules:

```yaml
groups:
- name: tool_alerts
  rules:
  - alert: HighToolErrorRate
    expr: rate(tool_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate for tool {{ $labels.tool_name }}"
```

## 🚀 Production Best Practices

### 1. Error Handling

Always handle errors gracefully:

```go
func (t *YourTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // Validate inputs
    if err := t.validateArgs(args); err != nil {
        return "", fmt.Errorf("invalid arguments: %w", err)
    }
    
    // Execute with timeout
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Implement with proper error handling
    result, err := t.doWork(ctx, args)
    if err != nil {
        slog.ErrorContext(ctx, "Tool execution failed",
            "tool", t.Name(),
            "error", err,
            "args", args,
        )
        return "", fmt.Errorf("execution failed: %w", err)
    }
    
    return result, nil
}
```

### 2. Logging

Use structured logging with context:

```go
func (t *YourTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    slog.InfoContext(ctx, "Tool execution started",
        "tool", t.Name(),
        "user_id", getUserID(ctx),
        "platform", getPlatform(ctx),
    )
    
    // ... tool logic
    
    slog.InfoContext(ctx, "Tool execution completed",
        "tool", t.Name(),
        "duration_ms", time.Since(start).Milliseconds(),
    )
    
    return result, nil
}
```

### 3. Performance Optimization

- Use connection pooling for external APIs
- Implement caching for expensive operations
- Use context for cancellation and timeouts
- Monitor memory usage and goroutine leaks

### 4. Security

- Validate all inputs
- Use prepared statements for database queries
- Implement rate limiting per user/tool
- Audit all sensitive operations

## 🔄 Versioning and Compatibility
