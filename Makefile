.PHONY: run fmt test test-integration lint build up down

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
	docker compose up -d db
	docker compose run --rm test go test -tags=integration ./tests/integration -count=1

build:
	go build -o backend-test ./cmd/server

up:
	docker compose up --build

down:
	docker compose down --remove-orphans
