package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	db, err := sqlx.Connect("postgres", "postgres://user:password@localhost:5430/uade?sslmode=disable")
	require.NoError(t, err)

	_, err = db.Exec(`TRUNCATE users RESTART IDENTITY CASCADE;`)
	require.NoError(t, err)

	return db
}

func TestRegisterAndLogin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{JWTSecret: "test-secret"}
	h := NewAuthHandler(db, cfg)

	// Register
	registerBody := `{"name":"TestUser","email":"user@example.com","password":"12345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(registerBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Register(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	// Login
	loginBody := `{"email":"user@example.com","password":"12345678"}`
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()

	h.Login(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}
