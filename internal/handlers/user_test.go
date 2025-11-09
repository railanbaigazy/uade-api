package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/stretchr/testify/require"
)

func setupDBUser(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", "postgres://user:password@localhost:5430/uade?sslmode=disable")
	require.NoError(t, err)
	if _, err := db.Exec(`TRUNCATE users RESTART IDENTITY CASCADE;`); err != nil {
		t.Fatalf("Failed to truncate users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (name, email, password_hash) VALUES ('Test','me@example.com','x');`); err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}
	return db
}

func TestProfileHandler(t *testing.T) {
	db := setupDBUser(t)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret"}
	h := NewUserHandler(db)

	// создаём тестовый токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": 1,
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(cfg.JWTSecret))

	req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	rec := httptest.NewRecorder()
	h.Profile(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var user map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	require.Equal(t, "me@example.com", user["email"])
}
