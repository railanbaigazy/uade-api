package app

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

func setupTestApp(t *testing.T) *http.ServeMux {
	db, err := sqlx.Connect("postgres", "postgres://user:password@localhost:5430/uade?sslmode=disable")
	require.NoError(t, err)

	cfg := &config.Config{JWTSecret: "test-secret"}
	a := New(db, cfg)
	return a.SetupRoutes()
}

func TestSetupRoutes(t *testing.T) {
	mux := setupTestApp(t)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"healthz works", http.MethodGet, "/healthz", "", http.StatusOK},
		{"register endpoint", http.MethodPost, "/api/auth/register", `{"name":"Tester","email":"t1@example.com","password":"123456"}`, http.StatusCreated},
		{"login endpoint", http.MethodPost, "/api/auth/login", `{"email":"t1@example.com","password":"123456"}`, http.StatusOK},
		{"unauthorized /me", http.MethodGet, "/api/users/me", "", http.StatusUnauthorized},
		{"unknown route", http.MethodGet, "/notfound", "", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			t.Logf("[%s] %s -> %d", tt.method, tt.path, rec.Code)
			// allow register to return 201 (created) or 409 (email already exists) in case tests rerun
			if tt.path == "/api/auth/register" {
				if !(rec.Code == http.StatusCreated || rec.Code == http.StatusConflict) {
					require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
				}
			} else {
				require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}
