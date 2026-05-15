# Backend Test Golang

A small Go HTTP service for the backend test assignment. It fetches Skinport item prices with in-memory caching and debits user balances in Postgres without an ORM.

## Requirements

- Go 1.25+
- Docker and Docker Compose
- Postgres 16 if running without Docker Compose

## Environment

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/backend_test?sslmode=disable` | Postgres connection DSN |
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `SKINPORT_BASE_URL` | `https://api.skinport.com/v1` | Skinport API base URL |
| `SKINPORT_CACHE_TTL` | `5m` | In-memory cache TTL for `/items` |
| `SKINPORT_TIMEOUT` | `10s` | Timeout for Skinport HTTP requests |
| `RUN_MIGRATIONS` | `true` | Run SQL migrations on application startup |

See `.env.example` for example values.

## Run

```bash
docker compose up --build
```

The service will be available at `http://localhost:8080`.
Postgres is published on `localhost:5432`. The application reaches it inside the Docker network as `db:5432`.

Run locally without Docker:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/backend_test?sslmode=disable'
go run ./cmd/server
```

## API

### Healthcheck

```bash
curl http://localhost:8080/healthz
```

Response:

```json
{"status":"ok"}
```

### Skinport items

```bash
curl 'http://localhost:8080/items?app_id=730&currency=USD'
```

The `app_id` and `currency` query parameters are optional. Defaults are `730` and `EUR`.

The endpoint performs two Skinport `/v1/items` requests, one with `tradable=1` and one with `tradable=0`. It merges items by `market_hash_name` and returns both minimum prices:

```json
{
  "items": [
    {
      "market_hash_name": "AK-47 | Aquamarine Revenge (Battle-Scarred)",
      "currency": "USD",
      "suggested_price": "13.18",
      "item_page": "https://skinport.com/item/...",
      "market_page": "https://skinport.com/market/...",
      "quantity": 25,
      "tradable_min_price": "11.33",
      "non_tradable_min_price": "10.90",
      "tradable_quantity": 25,
      "non_tradable_quantity": 4,
      "skinport_created_at": 1535988253,
      "skinport_updated_at": 1568073728
    }
  ]
}
```

### Debit user balance

```bash
curl -X POST http://localhost:8080/users/1/debit \
  -H 'Content-Type: application/json' \
  -d '{"amount":"100.00"}'
```

Response:

```json
{
  "id": 1,
  "user_id": 1,
  "amount": "100.00",
  "balance_before": "1000.00",
  "balance_after": "900.00",
  "created_at": "2026-05-15T12:00:00Z"
}
```

The balance cannot become negative. Debit is executed in a transaction with `SELECT ... FOR UPDATE`, and each operation is written to `balance_transactions`.

## Database

Migrations are embedded into the application and run on startup when `RUN_MIGRATIONS=true`.

Created tables:

- `users(id, balance)`
- `balance_transactions(id, user_id, amount, balance_before, balance_after, created_at)`

Seed data: user `id=1` with balance `1000.00`.

## Tests

Unit and HTTP tests:

```bash
go test ./...
```

Format:

```bash
make fmt
```

Lint:

```bash
make lint
```

Integration tests with Postgres:

```bash
make test-integration
```

Integration tests live in `tests/integration` behind the `integration` build tag. They use `TEST_DATABASE_URL`, cover both HTTP business routes, and verify concurrent debits so that the balance cannot go below zero.
`make test-integration` uses a separate Docker Postgres service, `test-db`, published on `localhost:5433`, so it does not touch the development database on `localhost:5432`.

Manual integration run:

```bash
docker compose --profile test up -d test-db
TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5433/backend_test?sslmode=disable' go test -tags=integration ./tests/integration -count=1
```
