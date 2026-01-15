.PHONY: dev build web run test vet clean

# Development: build web assets and run server
dev: web run

# Production build
build: web
	go build -o bin/server ./cmd/server

# Build web assets
web:
	cd web && bun install && bun run build

# Run the server
run:
	go run ./cmd/server/

# Run tests
test:
	go run ./tests/...

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf web/dist
