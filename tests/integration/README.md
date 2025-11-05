# Integration Tests

Integration tests verify interactions between components and external dependencies like databases and APIs.

## Characteristics

- **Real external dependencies** (databases, Redis, etc.)
- **Testcontainers for isolation**
- **Slower execution** (seconds per test)
- **Focus on component integration**

## Running Integration Tests

```bash
# Run all integration tests
go test ./tests/integration/... -v

# Run with test tags
go test -tags=integration ./tests/integration/... -v

# Run specific package
go test ./tests/integration/mongodb/... -v
```

## Test Structure

- Use testcontainers for database isolation
- Test real database operations
- Verify API integrations
- Clean up test data after each test
