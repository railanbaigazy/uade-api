package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/railanbaigazy/uade-api/internal/config"
	"github.com/stretchr/testify/require"
)

func setupTestApp(t *testing.T) *http.ServeMux {
	os.Setenv("DATABASE_URL", "postgres://user:password@localhost:5430/uade?sslmode=disable")
	os.Setenv("JWT_SECRET", "test-secret-key")

	cfg := config.Load()

	db, err := sqlx.Connect("postgres", cfg.DBURL)
	require.NoError(t, err)

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

		{"register endpoint", http.MethodPost, "/api/auth/register",
			`{"name":"Tester","email":"t1@example.com","password":"123456"}`,
			http.StatusCreated,
		},
		{"login endpoint", http.MethodPost, "/api/auth/login",
			`{"email":"t1@example.com","password":"123456"}`,
			http.StatusOK,
		},

		{"unauthorized /me", http.MethodGet, "/api/users/me", "", http.StatusUnauthorized},
		{"unauthorized get posts", http.MethodGet, "/api/posts", "", http.StatusUnauthorized},
		{"unauthorized create post", http.MethodPost, "/api/posts", `{"title":"x"}`, http.StatusUnauthorized},
		{"unauthorized update post", http.MethodPut, "/api/posts/1", `{"title":"x"}`, http.StatusUnauthorized},
		{"unauthorized delete post", http.MethodDelete, "/api/posts/1", "", http.StatusUnauthorized},

		{"unauthorized get agreements", http.MethodGet, "/api/agreements", "", http.StatusUnauthorized},
		{"unauthorized get agreement by id", http.MethodGet, "/api/agreements/1", "", http.StatusUnauthorized},
		{"unauthorized create agreement", http.MethodPost, "/api/agreements",
			`{"post_id":1,"principal_amount":1000,"interest_rate":0.1,"due_date":"2026-12-31","payment_frequency":"one_time","number_of_payments":1}`,
			http.StatusUnauthorized,
		},
		{"unauthorized accept agreement", http.MethodPost, "/api/agreements/1/accept", "", http.StatusUnauthorized},
		{"unauthorized cancel agreement", http.MethodPost, "/api/agreements/1/cancel", "", http.StatusUnauthorized},
		{"unauthorized update contract", http.MethodPut, "/api/agreements/1/contract",
			`{"contract_url":"https://example.com/contract.pdf","contract_hash":"abc123"}`,
			http.StatusUnauthorized,
		},

		{"unknown route", http.MethodGet, "/notfound", "", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			t.Logf("[%s] %s -> %d", tt.method, tt.path, rec.Code)

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
