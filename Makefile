.PHONY: run fmt test test-integration test-down lint build up down

TEST_COMPOSE := docker compose -p backend-test-golang-test -f docker-compose.test.yml

run:
	go run ./cmd/server

fmt:
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/ndrewnee/backend-test-golang .; \
	else \
		go fmt ./...; \
	fi

test:
	go test ./...

lint:
	golangci-lint run ./...

test-integration:
	$(TEST_COMPOSE) up -d --wait --force-recreate test-db
	$(TEST_COMPOSE) run --rm test go test -tags=integration ./tests/integration -count=1

test-down:
	$(TEST_COMPOSE) down --remove-orphans

build:
	go build -o backend-test ./cmd/server

up:
	docker compose up --build

down:
	docker compose down --remove-orphans
