# Uade API

**Uade API** is a monolithic Go REST API for an online borrowing–lending platform where users can make agreements, confirm money transfers and receive automatic reminders.

## What's in this repo

- Go REST API with JWT-protected endpoints
- PostgreSQL for persistence and RabbitMQ for async notifications
- Background worker for notifications
- Prometheus metrics endpoint
- OpenAPI 3.0 spec and Swagger UI

## API Docs (Swagger / OpenAPI)

Start the API, then open:

- Swagger UI: http://localhost:8080/docs/

The sources live in `docs/index.html` and `docs/openapi.yaml`.

## Useful endpoints

- Health check: `GET /healthz`
- Metrics: `GET /metrics`

## Repo layout

- `cmd/api`: API entrypoint
- `cmd/worker`: background worker entrypoint
- `internal`: application code (handlers, middleware, config, etc.)
- `migrations`: database migrations
- `docs`: Swagger UI and OpenAPI spec

## Getting Started

### 1. Clone the repo

```bash
git clone https://github.com/railanbaigazy/uade-api.git
cd uade-api
```

### 2. Build containers

```bash
docker-compose up --build
```

API will be available at:

http://localhost:8080

### 3. Run migrations

```bash
migrate -path migrations -database "postgres://user:password@localhost:5432/db?sslmode=disable" up
```

### 4. Run tests

```bash
make test
```

## Development

### Setting up environment

Create a .env file from template

Content of `.env`:

```
DATABASE_URL=postgres://user:password@localhost:5430/uade?sslmode=disable
JWT_SECRET=your-super-secret-key-change-in-production
APP_ENV=development
PORT=8080
AMQP_URL=amqp://guest:guest@localhost:5672/
```

### Starting locally without Docker:

```bash
make run
```

### Testing locally

```bash
make test
```

### Formatting code

```bash
make format
```

### Running only PostgreSQL with Docker:

```bash
docker-compose up db
```
