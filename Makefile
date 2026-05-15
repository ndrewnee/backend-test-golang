.PHONY: run test test-integration lint build up down

run:
	go run ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run ./...

test-integration:
	docker compose up -d db
	docker compose run --rm test go test -tags=integration ./... -run Integration -count=1

build:
	go build -o backend-test ./cmd/server

up:
	docker compose up --build

down:
	docker compose down --remove-orphans
