# WealthPath Backend

Go API server for the WealthPath personal finance application.

## Structure

```
├── cmd/api/          # Application entrypoint
├── internal/         # Private application code
│   ├── config/       # Configuration
│   ├── handler/      # HTTP handlers
│   ├── model/        # Data models
│   ├── repository/   # Database access
│   └── service/      # Business logic
├── pkg/              # Public packages
├── migrations/       # Flyway SQL migrations
├── docs/             # Swagger documentation
└── Dockerfile        # Container build
```

## Development

```bash
# Install dependencies
go mod download

# Run tests
go test -v ./...

# Run locally
go run cmd/api/main.go

# Build
go build -o bin/api cmd/api/main.go
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | Secret for JWT signing |
| `PORT` | Server port (default: 8080) |
| `ALLOWED_ORIGINS` | CORS allowed origins |

## Docker

```bash
# Build backend image
docker build -t wealthpath-backend .

# Build migrations image
docker build -t wealthpath-migrations -f migrations/Dockerfile migrations/
```

## API Documentation

Swagger documentation is available at `/swagger/` when running locally.

## CI/CD

On push to `main`:
1. Runs tests
2. Builds Docker images
3. Pushes to GitHub Container Registry:
   - `ghcr.io/wealthpathorganization/backend:latest`
   - `ghcr.io/wealthpathorganization/migrations:latest`
