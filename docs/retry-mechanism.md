# Retry Mechanism with Exponential Backoff

## Overview

The retry mechanism provides robust error handling for external API calls with exponential backoff and jitter to handle transient failures gracefully.

## Configuration

### Environment Variables

```bash
# Retry Configuration
RETRY_MAX_ATTEMPTS=3        # Maximum number of retry attempts (default: 3)
RETRY_BASE_DELAY_MS=500     # Base delay between retries in milliseconds (default: 500ms)
RETRY_MAX_DELAY_MS=5000     # Maximum delay between retries in milliseconds (default: 5s)
```

### Default Configuration

```go
RetryConfig{
    MaxAttempts: 3,
    BaseDelay:   500 * time.Millisecond,
    MaxDelay:    5 * time.Second,
}
```

## Usage

### Basic Retry

```go
import "github.com/8adimka/Go_AI_Assistant/internal/retry"

err := retry.Retry(ctx, config, func() error {
    return someAPICall()
})
```

### Retry with Result

```go
result, err := retry.RetryWithResult(ctx, config, func() (ResultType, error) {
    return someAPICallWithResult()
})
```

### Using Application Configuration

```go
import (
    "github.com/8adimka/Go_AI_Assistant/internal/config"
    "github.com/8adimka/Go_AI_Assistant/internal/retry"
)

cfg := config.Load()
retryConfig := retry.ConfigFromAppConfig(cfg)
```

## Retry Strategy

### Retryable Errors

The system automatically retries on:

- **5xx HTTP status codes** (server errors)
- **429 Too Many Requests** (rate limiting)
- **Network timeouts** and connection issues
- **Context deadline exceeded** errors

### Non-Retryable Errors

The system does NOT retry on:

- **4xx HTTP status codes** (client errors)
- **Invalid requests** (bad parameters, authentication errors)
- **Business logic errors**

### Exponential Backoff with Jitter

The retry delay follows exponential backoff with random jitter:

```
delay = base_delay * (2^attempt) * random(0.5, 1.5)
```

Example delays for base delay of 500ms:

- Attempt 1: 500ms - 1500ms
- Attempt 2: 1000ms - 3000ms  
- Attempt 3: 2000ms - 6000ms (capped at max delay)

## Integration Points

### OpenAI API

The retry mechanism is integrated into:

- `Assistant.Title()` - Title generation
- `Assistant.Reply()` - Conversation replies

### WeatherAPI

The retry mechanism is integrated into:

- `WeatherAPIClient.GetCurrent()` - Current weather data
- `WeatherAPIClient.GetForecast()` - Weather forecast

## Circuit Breaker Compatibility

The retry mechanism works alongside the existing circuit breaker pattern:

1. **Retry** handles transient failures with exponential backoff
2. **Circuit Breaker** prevents cascading failures when services are down
3. **Rate Limiting** prevents overwhelming external APIs

## Observability

### Logging

The retry mechanism logs:

- Retry attempts with attempt number and delay
- Non-retryable errors (warn level)
- Max retry attempts reached (warn level)

### Metrics (Planned)

Future metrics integration will track:

- `api_retry_attempts_total` - Total retry attempts per service
- `api_retry_failures_total` - Total retry failures per service  
- `api_retry_success_total` - Total successful retries per service

## Best Practices

1. **Configure appropriately** for your use case:
   - High-traffic services: lower max attempts (2-3)
   - Critical services: higher max attempts (3-5)

2. **Monitor retry patterns** to identify persistent issues

3. **Combine with circuit breakers** for comprehensive resilience

4. **Test failure scenarios** to ensure graceful degradation
