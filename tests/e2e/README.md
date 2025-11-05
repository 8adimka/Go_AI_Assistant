# End-to-End (E2E) Tests

E2E tests verify complete user workflows from API calls to external service integrations.

## Characteristics

- **Full system integration** with all external services
- **Real API calls** to OpenAI, WeatherAPI, etc.
- **Slowest execution** (minutes per test suite)
- **Focus on user scenarios**

## Running E2E Tests

```bash
# Run all E2E tests (requires external API keys)
go test ./tests/e2e/... -v

# Run with test tags
go test -tags=e2e ./tests/e2e/... -v

# Run specific scenario
go test ./tests/e2e/conversation/... -v
```

## Prerequisites

- Valid OpenAI API key
- Valid WeatherAPI key (optional)
- Running Redis and MongoDB instances

## Test Structure

- Test complete user workflows
- Verify real external API integrations
- Use real configuration and environment
- Focus on critical user journeys
