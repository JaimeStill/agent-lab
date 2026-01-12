# Go Commands

## Validation

```bash
go vet ./...  # Check for errors (NOT go build)
```

## Testing

```bash
go test ./tests/...  # Run all tests
```

## Run

```bash
go run ./cmd/server  # Start the web service
```

## Migrations

```bash
# Apply migrations
go run ./cmd/migrate -dsn "postgres://agent_lab:agent_lab@localhost:5432/agent_lab?sslmode=disable" -up

# Rollback one migration
go run ./cmd/migrate -dsn "postgres://agent_lab:agent_lab@localhost:5432/agent_lab?sslmode=disable" -down
```

## Seed Data

```bash
go run ./cmd/seed -dsn "postgres://agent_lab:agent_lab@localhost:5432/agent_lab?sslmode=disable" -all
```
