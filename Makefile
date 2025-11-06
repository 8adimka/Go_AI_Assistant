.PHONY: test-unit test-integration test-e2e test-all

# Run all unit tests
test-unit:
	go test ./tests/unit/...

# Run all integration tests
test-integration:
	go test ./tests/integration/...

# Run all e2e tests
test-e2e:
	go test ./tests/e2e/...

# Run all tests
test-all: test-unit test-integration test-e2e

# Run specific unit test packages
test-unit-circuitbreaker:
	go test ./tests/unit/circuitbreaker

test-unit-chat:
	go test ./tests/unit/chat

test-unit-errorsx:
	go test ./tests/unit/errorsx

test-unit-health:
	go test ./tests/unit/health

test-unit-httpx:
	go test ./tests/unit/httpx

test-unit-metrics:
	go test ./tests/unit/metrics

test-unit-redisx:
	go test ./tests/unit/redisx

test-unit-tools:
	go test ./tests/unit/tools

# Clean test cache
test-clean:
	go clean -testcache

# Show test coverage
test-coverage:
	go test -coverprofile=coverage.out ./tests/unit/...
	go tool cover -html=coverage.out -o coverage.html

# Help
help:
	@echo "Available targets:"
	@echo "  test-unit              - Run all unit tests"
	@echo "  test-integration       - Run all integration tests"
	@echo "  test-e2e               - Run all e2e tests"
	@echo "  test-all               - Run all tests"
	@echo "  test-unit-circuitbreaker - Run circuitbreaker unit tests"
	@echo "  test-unit-chat         - Run chat unit tests"
	@echo "  test-unit-errorsx      - Run errorsx unit tests"
	@echo "  test-unit-health       - Run health unit tests"
	@echo "  test-unit-httpx        - Run httpx unit tests"
	@echo "  test-unit-metrics      - Run metrics unit tests"
	@echo "  test-unit-redisx       - Run redisx unit tests"
	@echo "  test-unit-tools        - Run tools unit tests"
	@echo "  test-clean             - Clean test cache"
	@echo "  test-coverage          - Generate test coverage report"
	@echo "  help                   - Show this help message"
