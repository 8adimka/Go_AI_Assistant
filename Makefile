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
