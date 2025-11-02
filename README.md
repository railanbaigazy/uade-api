# Uade API

**Uade API** is a monolithic Go REST API for an online borrowing–lending platform where users can make agreements, confirm money transfers and receive automatic reminders.

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

Create a .env file:

```
DATABASE_URL=postgres://user:password@localhost:5430/uade?sslmode=disable
PORT=8080
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
