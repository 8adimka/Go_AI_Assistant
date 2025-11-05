gen:
	protoc --proto_path=. --twirp_out=. --go_out=. rpc/*.proto

run:
	go run ./cmd/server

# Test commands
test:
	go test ./...

test-unit:
	go test ./tests/unit/... -v

test-integration:
	go test -tags=integration ./tests/integration/... -v

test-e2e:
	go test -tags=e2e ./tests/e2e/... -v

test-performance:
	go test -bench=. ./tests/performance/... -benchmem

test-all: test-unit test-integration test-e2e test-performance

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

up:
	docker compose up -d

down:
	docker compose down

# Database migration commands
migrate-up:
	@echo "Running migrations..."
	@mongosh $${MONGO_URI:-mongodb://acai:travel@localhost:27017/acai} --file migrations/000001_init.up.js

migrate-down:
	@echo "Rolling back migrations..."
	@mongosh $${MONGO_URI:-mongodb://acai:travel@localhost:27017/acai} --file migrations/000001_init.down.js

migrate-status:
	@echo "Checking indexes..."
	@mongosh $${MONGO_URI:-mongodb://acai:travel@localhost:27017/acai} --eval "db.conversations.getIndexes(); db.messages.getIndexes();"

# Database backup and restore
backup:
	@echo "Creating backup..."
	@mkdir -p backups
	@mongodump --uri="$${MONGO_URI:-mongodb://acai:travel@localhost:27017}" --db=acai --out=backups/backup-$$(date +%Y%m%d-%H%M%S)

restore:
	@echo "Restoring from backup..."
	@mongorestore --uri="$${MONGO_URI:-mongodb://acai:travel@localhost:27017}" --db=acai $(BACKUP_PATH)

# Smoke test
smoke:
	@chmod +x scripts/smoke-test.sh
	@./scripts/smoke-test.sh
