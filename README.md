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

Create a .env file from template:

```bash
cp .env.example .env
```

Content of `.env`:

```
DATABASE_URL=postgres://user:password@localhost:5430/uade?sslmode=disable
JWT_SECRET=your-super-secret-key-change-in-production
APP_ENV=development
PORT=8080
```

### Starting locally without Docker:

```bash
make run
```

## 🔐 JWT Authentication

This branch implements JWT-based authentication:

### Register new user (POST /api/auth/register)

```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "securePassword123"
  }'
```

**Response:** 201 Created

### Login (POST /api/auth/login)

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "securePassword123"
  }'
```

**Response:** 200 OK

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Access protected endpoint (GET /api/users/me)

```bash
curl -X GET http://localhost:8080/api/users/me \
  -H "Authorization: Bearer <your-jwt-token>"
```

**Response:** 200 OK

```json
{
  "id": 1,
  "name": "John Doe",
  "email": "john@example.com",
  "role": "user",
  "state": "active",
  "created_at": "2025-11-11T12:34:56Z"
}
```

### Security Features

- ✅ Passwords hashed with bcrypt
- ✅ JWT tokens (HS256, 24-hour expiry)
- ✅ Email validation
- ✅ Request validation
- ✅ Error handling for edge cases
- ✅ Unique email constraint

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
