.PHONY: help

# Variables
BINARY_NAME=go-ai-assistant
MAIN_PATH=./cmd/server
BACKUP_DIR=backups
TIMESTAMP=$(shell date +%Y%m%d-%H%M%S)

# ============================================================================
# HELP
# ============================================================================

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
	awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ============================================================================
# DEVELOPMENT
# ============================================================================

up: ## Start Docker services (MongoDB + Redis)
	docker-compose up -d
	@echo "✓ Services started"

down: ## Stop Docker services
	docker-compose down
	@echo "✓ Services stopped"

run: ## Run the application
	go run $(MAIN_PATH)/main.go

build: ## Build the application binary
	go build -o $(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "✓ Binary built: $(BINARY_NAME)"

fmt: ## Format Go code
	gofmt -w .
	@echo "✓ Code formatted"

lint: ## Run golangci-lint
	golangci-lint run ./...
	@echo "✓ Linting complete"

vet: ## Run go vet
	go vet ./...
	@echo "✓ Static analysis complete"

# ============================================================================
# TESTING
# ============================================================================

test: test-all ## Alias for test-all

test-all: test-unit test-integration test-e2e ## Run all tests
	@echo "✓ All tests passed"

test-unit: ## Run unit tests
	go test ./tests/unit/...
	@echo "✓ Unit tests passed"

test-integration: ## Run integration tests
	go test ./tests/integration/...
	@echo "✓ Integration tests passed"

test-e2e: ## Run end-to-end tests
	go test -tags=e2e ./tests/e2e/...
	@echo "✓ E2E tests passed"

test-performance: ## Run performance benchmarks
	go test -bench=. -benchmem ./tests/performance/...
	@echo "✓ Performance tests complete"

test-coverage: ## Generate test coverage report
	go test -coverprofile=coverage.out ./tests/unit/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

test-clean: ## Clean test cache
	go clean -testcache
	@echo "✓ Test cache cleaned"

smoke: ## Run smoke tests (requires running server)
	@chmod +x scripts/smoke-test.sh
	@./scripts/smoke-test.sh
	@echo "✓ Smoke tests passed"

# ============================================================================
# DATABASE
# ============================================================================

migrate-up: ## Apply database migrations
	@echo "Creating MongoDB indexes..."
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.createIndex({user_id: 1, created_at: -1})" --quiet || true
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.createIndex({platform: 1, chat_id: 1})" --quiet || true
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.createIndex({is_active: 1, last_activity: -1})" --quiet || true
	@echo "✓ Migrations applied"

migrate-down: ## Rollback database migrations
	@echo "Dropping MongoDB indexes..."
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.dropIndex('user_id_1_created_at_-1')" --quiet 2>/dev/null || true
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.dropIndex('platform_1_chat_id_1')" --quiet 2>/dev/null || true
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.dropIndex('is_active_1_last_activity_-1')" --quiet 2>/dev/null || true
	@echo "✓ Migrations rolled back"

migrate-status: ## Show database migration status
	@echo "MongoDB Indexes:"
	@docker exec mongodb mongosh mongodb://acai:travel@localhost:27017/acai \
		--eval "db.conversations.getIndexes()" --quiet

backup: ## Create database backup
	@mkdir -p $(BACKUP_DIR)/backup-$(TIMESTAMP)
	@echo "Creating backup..."
	@docker exec mongodb mongodump --uri="mongodb://acai:travel@localhost:27017" \
		--out=/tmp/backup-$(TIMESTAMP) --quiet
	@docker cp mongodb:/tmp/backup-$(TIMESTAMP) $(BACKUP_DIR)/
	@echo "✓ Backup created: $(BACKUP_DIR)/backup-$(TIMESTAMP)"

restore: ## Restore database from backup (usage: make restore BACKUP_PATH=backups/backup-20241106-120000/acai)
	@if [ -z "$(BACKUP_PATH)" ]; then \
		echo "Error: BACKUP_PATH is required"; \
		echo "Usage: make restore BACKUP_PATH=backups/backup-YYYYMMDD-HHMMSS/acai"; \
		exit 1; \
	fi
	@echo "Restoring from $(BACKUP_PATH)..."
	@docker cp $(BACKUP_PATH) mongodb:/tmp/restore
	@docker exec mongodb mongorestore --uri="mongodb://acai:travel@localhost:27017" \
		--drop /tmp/restore --quiet
	@echo "✓ Backup restored from $(BACKUP_PATH)"

# ============================================================================
# CODE QUALITY
# ============================================================================

check: fmt vet lint ## Run all code quality checks
	@echo "✓ All checks passed"

mod-tidy: ## Tidy go modules
	go mod tidy
	@echo "✓ Modules tidied"

mod-verify: ## Verify go modules
	go mod verify
	@echo "✓ Modules verified"

# ============================================================================
# UTILITIES
# ============================================================================

clean: test-clean ## Clean build artifacts and caches
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "✓ Cleaned"

logs: ## Show Docker logs
	docker-compose logs -f

ps: ## Show running Docker containers
	docker-compose ps
