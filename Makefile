.PHONY: run build build-ci test migrate clean

# Run the server
run:
	go run cmd/api/main.go

# Build the binary
# Uses /tmp for Go caches to avoid permission issues in CI/CD environments
build:
	@mkdir -p bin
	GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod go build -o bin/api cmd/api/main.go

# Build for CI/CD environments (with explicit cache directories and CGO disabled)
# This is the recommended target for automated build agents
build-ci:
	@mkdir -p bin /tmp/go-cache /tmp/go-mod
	GOCACHE=/tmp/go-cache GOMODCACHE=/tmp/go-mod CGO_ENABLED=0 go build -o bin/api cmd/api/main.go

# Clean build artifacts and cache
clean:
	@rm -rf bin/
	@echo "Build artifacts cleaned"

# Run tests
test:
	go test -v ./...

# Run migrations
migrate:
	psql -d wealthpath -f migrations/001_initial.sql

# Download dependencies
deps:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Generate Swagger documentation
swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal --packageName docs




