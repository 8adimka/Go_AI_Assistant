# Unit Tests

Unit tests are fast, isolated tests that verify individual components without external dependencies.

## Characteristics

- **No external dependencies** (databases, APIs, network calls)
- **Fast execution** (milliseconds per test)
- **Mock all external interfaces**
- **Focus on business logic**

## Running Unit Tests

```bash
# Run all unit tests
go test ./tests/unit/... -v

# Run with coverage
go test ./tests/unit/... -cover

# Run specific package
go test ./tests/unit/assistant/... -v
```

## Test Structure

- Mock external dependencies
- Test edge cases and error scenarios
- Focus on pure logic and transformations
